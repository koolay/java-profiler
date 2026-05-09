package com.ebpfjava.examples.httpdemo;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import java.io.IOException;
import java.io.OutputStream;
import java.lang.management.ManagementFactory;
import java.net.HttpURLConnection;
import java.net.InetSocketAddress;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.time.Instant;
import java.util.HashMap;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.ThreadFactory;
import java.util.concurrent.atomic.AtomicInteger;

public final class DemoHttpService {
  private static final Object LOCK_MONITOR = new Object();
  private static final int DEFAULT_PORT = 8080;
  private static final int MAX_DURATION_MS = 30_000;

  private DemoHttpService() {}

  public static void main(String[] args) throws IOException {
    int port = readPort();
    int workers = readIntEnv("DEMO_WORKERS", Runtime.getRuntime().availableProcessors() * 2);
    ExecutorService executor = Executors.newFixedThreadPool(Math.max(2, workers), namedThreads());
    HttpServer server = createServer(new InetSocketAddress("0.0.0.0", port), executor);

    Runtime.getRuntime()
        .addShutdownHook(
            new Thread(
                () -> {
                  server.stop(1);
                  executor.shutdownNow();
                },
                "shutdown"));

    server.start();
    System.out.println("jdk17-http-demo listening on :" + port);
  }

  static HttpServer createServer(InetSocketAddress address, ExecutorService executor)
      throws IOException {
    Objects.requireNonNull(address, "address");
    Objects.requireNonNull(executor, "executor");

    HttpServer server = HttpServer.create(address, 64);
    server.setExecutor(executor);
    server.createContext("/health", DemoHttpService::handleHealth);
    server.createContext("/work", DemoHttpService::handleWork);
    server.createContext("/threads", DemoHttpService::handleThreads);
    server.createContext("/", DemoHttpService::handleIndex);
    return server;
  }

  private static void handleHealth(HttpExchange exchange) throws IOException {
    if (!requireGet(exchange)) {
      return;
    }

    writeJson(
        exchange,
        HttpURLConnection.HTTP_OK,
        "{"
            + "\"status\":\"ok\","
            + "\"service\":\"jdk17-http-demo\","
            + "\"runtime\":\"jdk17\","
            + "\"jvm\":\""
            + escape(ManagementFactory.getRuntimeMXBean().getVmName())
            + "\""
            + "}");
  }

  private static void handleWork(HttpExchange exchange) throws IOException {
    if (!requireGet(exchange)) {
      return;
    }

    Map<String, String> query = parseQuery(exchange.getRequestURI().getRawQuery());
    String mode = query.getOrDefault("mode", "cpu").toLowerCase(Locale.ROOT);
    int durationMs = clamp(readInt(query.get("durationMs"), 1_000), 1, MAX_DURATION_MS);
    Instant startedAt = Instant.now();
    long operations;

    switch (mode) {
      case "cpu" -> operations = burnCpu(durationMs);
      case "alloc" -> operations = allocateObjects(durationMs);
      case "lock" -> operations = contendLock(durationMs);
      default -> {
        writeJson(
            exchange,
            HttpURLConnection.HTTP_BAD_REQUEST,
            "{\"error\":\"mode must be one of cpu, alloc, lock\"}");
        return;
      }
    }

    long elapsedMs = Math.max(1, Duration.between(startedAt, Instant.now()).toMillis());
    writeJson(
        exchange,
        HttpURLConnection.HTTP_OK,
        "{"
            + "\"mode\":\""
            + mode
            + "\","
            + "\"durationMs\":"
            + elapsedMs
            + ","
            + "\"operations\":"
            + operations
            + ","
            + "\"thread\":\""
            + escape(Thread.currentThread().getName())
            + "\""
            + "}");
  }

  private static void handleThreads(HttpExchange exchange) throws IOException {
    if (!requireGet(exchange)) {
      return;
    }

    Map<String, String> query = parseQuery(exchange.getRequestURI().getRawQuery());
    int durationMs = clamp(readInt(query.get("durationMs"), 5_000), 1, MAX_DURATION_MS);
    Thread sleeper =
        new Thread(
            () -> {
              try {
                Thread.sleep(durationMs);
              } catch (InterruptedException interrupted) {
                Thread.currentThread().interrupt();
              }
            },
            "demo-sleeping-worker");
    Thread blocked =
        new Thread(
            () -> {
              synchronized (LOCK_MONITOR) {
                // Immediately exits after the request thread releases the monitor.
              }
            },
            "demo-blocked-worker");

    sleeper.start();
    synchronized (LOCK_MONITOR) {
      blocked.start();
      sleepQuietly(Math.min(durationMs, 1_000));
    }

    writeJson(
        exchange,
        HttpURLConnection.HTTP_OK,
        "{"
            + "\"status\":\"started\","
            + "\"durationMs\":"
            + durationMs
            + ","
            + "\"threads\":[\"demo-sleeping-worker\",\"demo-blocked-worker\"]"
            + "}");
  }

  private static void handleIndex(HttpExchange exchange) throws IOException {
    if (!requireGet(exchange)) {
      return;
    }

    writeJson(
        exchange,
        HttpURLConnection.HTTP_OK,
        "{"
            + "\"service\":\"jdk17-http-demo\","
            + "\"endpoints\":[\"/health\",\"/work?mode=cpu|alloc|lock&durationMs=1000\","
            + "\"/threads?durationMs=5000\"]"
            + "}");
  }

  private static long burnCpu(int durationMs) {
    long deadline = System.nanoTime() + Duration.ofMillis(durationMs).toNanos();
    long operations = 0;
    double value = 0.42;
    while (System.nanoTime() < deadline) {
      value += Math.sqrt(value + operations % 97);
      operations++;
    }
    return operations + (long) value;
  }

  private static long allocateObjects(int durationMs) {
    long deadline = System.nanoTime() + Duration.ofMillis(durationMs).toNanos();
    long operations = 0;
    while (System.nanoTime() < deadline) {
      byte[][] chunks = new byte[128][];
      for (int i = 0; i < chunks.length; i++) {
        chunks[i] = new byte[8 * 1024];
        operations++;
      }
    }
    return operations;
  }

  private static long contendLock(int durationMs) {
    long deadline = System.nanoTime() + Duration.ofMillis(durationMs).toNanos();
    long operations = 0;
    while (System.nanoTime() < deadline) {
      synchronized (LOCK_MONITOR) {
        sleepQuietly(2);
        operations++;
      }
    }
    return operations;
  }

  private static boolean requireGet(HttpExchange exchange) throws IOException {
    if ("GET".equals(exchange.getRequestMethod())) {
      return true;
    }
    exchange.getResponseHeaders().set("Allow", "GET");
    writeJson(exchange, HttpURLConnection.HTTP_BAD_METHOD, "{\"error\":\"method not allowed\"}");
    return false;
  }

  private static void writeJson(HttpExchange exchange, int statusCode, String body)
      throws IOException {
    byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
    exchange.getResponseHeaders().set("Content-Type", "application/json; charset=utf-8");
    exchange.sendResponseHeaders(statusCode, bytes.length);
    try (OutputStream output = exchange.getResponseBody()) {
      output.write(bytes);
    }
  }

  private static Map<String, String> parseQuery(String rawQuery) {
    Map<String, String> values = new HashMap<>();
    if (rawQuery == null || rawQuery.isBlank()) {
      return values;
    }

    for (String pair : rawQuery.split("&")) {
      int separator = pair.indexOf('=');
      String key = separator >= 0 ? pair.substring(0, separator) : pair;
      String value = separator >= 0 ? pair.substring(separator + 1) : "";
      values.put(decode(key), decode(value));
    }
    return values;
  }

  private static String decode(String value) {
    return URLDecoder.decode(value, StandardCharsets.UTF_8);
  }

  private static String escape(String value) {
    return value.replace("\\", "\\\\").replace("\"", "\\\"");
  }

  private static int readPort() {
    return clamp(readIntEnv("PORT", DEFAULT_PORT), 1, 65_535);
  }

  private static int readIntEnv(String name, int fallback) {
    return readInt(System.getenv(name), fallback);
  }

  private static int readInt(String value, int fallback) {
    if (value == null || value.isBlank()) {
      return fallback;
    }
    try {
      return Integer.parseInt(value);
    } catch (NumberFormatException ignored) {
      return fallback;
    }
  }

  private static int clamp(int value, int min, int max) {
    return Math.max(min, Math.min(max, value));
  }

  private static void sleepQuietly(long millis) {
    try {
      Thread.sleep(millis);
    } catch (InterruptedException interrupted) {
      Thread.currentThread().interrupt();
    }
  }

  private static ThreadFactory namedThreads() {
    AtomicInteger nextId = new AtomicInteger(1);
    return runnable -> {
      Thread thread = new Thread(runnable, "demo-http-worker-" + nextId.getAndIncrement());
      thread.setDaemon(false);
      return thread;
    };
  }
}
