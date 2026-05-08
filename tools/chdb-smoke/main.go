package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chdb-io/chdb-go/chdb"
)

func main() {
	schemaPath := "../../db/clickhouse/001_initial_profile_schema.sql"
	if len(os.Args) > 1 {
		schemaPath = os.Args[1]
	}
	if err := run(schemaPath); err != nil {
		fmt.Fprintf(os.Stderr, "chDB smoke verification failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("chDB smoke verification passed")
}

func run(schemaPath string) error {
	if err := verifyStatelessQuery(); err != nil {
		return err
	}
	if err := verifyStatefulSession(); err != nil {
		return err
	}
	if err := verifyProfilerDDL(schemaPath); err != nil {
		return err
	}
	if err := verifyStreamingAPI(); err != nil {
		return err
	}
	return nil
}

func verifyStatelessQuery() error {
	result, err := chdb.Query("SELECT 1", "CSV")
	if err != nil {
		return fmt.Errorf("stateless Query failed: %w", err)
	}
	defer result.Free()
	if err := result.Error(); err != nil {
		return fmt.Errorf("stateless Query result error: %w", err)
	}
	if strings.TrimSpace(result.String()) != "1" {
		return fmt.Errorf("stateless Query returned %q, want 1", result.String())
	}
	return nil
}

func verifyStatefulSession() error {
	dir, err := os.MkdirTemp("", "java-profiler-chdb-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	session, err := chdb.NewSession(dir)
	if err != nil {
		return fmt.Errorf("NewSession failed: %w", err)
	}
	defer session.Close()
	statements := []string{
		"CREATE DATABASE IF NOT EXISTS smoke",
		"CREATE TABLE IF NOT EXISTS smoke.profile_samples (batch_id String, namespace LowCardinality(String), service LowCardinality(String), value UInt64) ENGINE = MergeTree ORDER BY (namespace, service, batch_id)",
		"INSERT INTO smoke.profile_samples VALUES ('batch-1', 'prod', 'checkout', 42)",
	}
	for _, statement := range statements {
		if err := execSession(session, statement); err != nil {
			return err
		}
	}
	result, err := session.Query("SELECT sum(value) FROM smoke.profile_samples WHERE namespace = 'prod' AND service = 'checkout'", "CSV")
	if err != nil {
		return fmt.Errorf("session query failed: %w", err)
	}
	defer result.Free()
	if err := result.Error(); err != nil {
		return fmt.Errorf("session query result error: %w", err)
	}
	if strings.TrimSpace(result.String()) != "42" {
		return fmt.Errorf("session query returned %q, want 42", result.String())
	}
	return nil
}

func verifyProfilerDDL(schemaPath string) error {
	absolute, err := filepath.Abs(schemaPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return fmt.Errorf("read schema %s: %w", absolute, err)
	}
	dir, err := os.MkdirTemp("", "java-profiler-chdb-schema-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	session, err := chdb.NewSession(dir)
	if err != nil {
		return fmt.Errorf("NewSession for schema failed: %w", err)
	}
	defer session.Close()
	for _, statement := range splitSQL(string(data)) {
		if err := execSession(session, statement); err != nil {
			return fmt.Errorf("schema statement failed: %w\n%s", err, statement)
		}
	}
	result, err := session.Query("SELECT count() FROM system.tables WHERE database = currentDatabase() AND startsWith(name, 'java_profiler_')", "CSV")
	if err != nil {
		return fmt.Errorf("system table query failed: %w", err)
	}
	defer result.Free()
	if err := result.Error(); err != nil {
		return fmt.Errorf("system table query result error: %w", err)
	}
	if strings.TrimSpace(result.String()) != "7" {
		return fmt.Errorf("schema created %q java_profiler tables, want 7", strings.TrimSpace(result.String()))
	}
	return nil
}

func verifyStreamingAPI() error {
	stream, err := chdb.QueryStream("SELECT number FROM numbers(3) ORDER BY number", "CSV")
	if err != nil {
		return fmt.Errorf("QueryStream failed: %w", err)
	}
	defer stream.Free()
	var output strings.Builder
	for {
		chunk := stream.GetNext()
		if chunk == nil {
			break
		}
		output.WriteString(chunk.String())
		chunk.Free()
	}
	if err := stream.Error(); err != nil {
		return fmt.Errorf("QueryStream result error: %w", err)
	}
	if strings.TrimSpace(output.String()) != "0\n1\n2" {
		return fmt.Errorf("QueryStream returned %q, want 0/1/2 rows", output.String())
	}
	return nil
}

func execSession(session *chdb.Session, statement string) error {
	result, err := session.Query(statement, "CSV")
	if err != nil {
		return fmt.Errorf("query %q failed: %w", statement, err)
	}
	defer result.Free()
	if err := result.Error(); err != nil {
		return fmt.Errorf("query %q result error: %w", statement, err)
	}
	return nil
}

func splitSQL(schema string) []string {
	parts := strings.Split(schema, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement == "" {
			continue
		}
		statements = append(statements, statement)
	}
	return statements
}
