package com.ebpfjava.threads;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertFalse;

class ThreadSnapshotCommandTest {
    @Test
    void capturesCurrentJvmThreads() {
        ThreadSnapshotPayload payload = ThreadSnapshotCommand.captureDefault(32);
        assertFalse(payload.threads.isEmpty());
    }
}
