package com.ebpfjava.threads;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class ThreadSnapshotPayloadTest {
    @Test
    void recordsThreadFields() {
        ThreadSnapshotPayload.ThreadRecord record = new ThreadSnapshotPayload.ThreadRecord();
        record.id = 7;
        record.name = "worker";
        record.stack.add("Example.run");
        assertEquals("worker", record.name);
        assertEquals(1, record.stack.size());
    }
}
