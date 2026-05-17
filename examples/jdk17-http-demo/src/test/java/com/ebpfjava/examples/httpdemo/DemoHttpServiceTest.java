package com.ebpfjava.examples.httpdemo;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import com.sun.net.httpserver.HttpServer;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

class DemoHttpServiceTest {
  private ExecutorService executor;
  private HttpServer server;
  private URI baseUri;
  private final HttpClient client =
      HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(2)).build();

  @BeforeEach
  void startServer() throws IOException {
    executor = Executors.newFixedThreadPool(4);
    server = DemoHttpService.createServer(new InetSocketAddress("127.0.0.1", 0), executor);
    server.start();
    baseUri = URI.create("http://127.0.0.1:" + server.getAddress().getPort());
  }

  @AfterEach
  void stopServer() {
    if (server != null) {
      server.stop(0);
    }
    if (executor != null) {
      executor.shutdownNow();
    }
  }

  @Test
  void healthEndpointReturnsServiceIdentity() throws Exception {
    HttpResponse<String> response = get("/health");

    assertEquals(200, response.statusCode());
    assertTrue(response.body().contains("\"status\":\"ok\""));
    assertTrue(response.body().contains("\"runtime\":\"jdk17\""));
  }

  @Test
  void workEndpointCanExerciseProfilerEvidencePaths() throws Exception {
    for (String mode : new String[] {"cpu", "alloc", "gc", "io", "wall", "lock"}) {
      HttpResponse<String> response = get("/work?mode=" + mode + "&durationMs=20");

      assertEquals(200, response.statusCode(), mode);
      assertTrue(response.body().contains("\"mode\":\"" + mode + "\""), mode);
      assertTrue(response.body().contains("\"durationMs\":"), mode);
    }
  }

  private HttpResponse<String> get(String path) throws Exception {
    HttpRequest request =
        HttpRequest.newBuilder(baseUri.resolve(path)).timeout(Duration.ofSeconds(5)).GET().build();
    return client.send(request, HttpResponse.BodyHandlers.ofString());
  }
}
