package com.ebpfjava.threads;

import java.util.ArrayList;
import java.util.List;

public final class ThreadSnapshotPayload {
    public long capturedAtMillis;
    public boolean threadCpuTimeSupported;
    public boolean contentionMonitoringEnabled;
    public List<ThreadRecord> threads = new ArrayList<>();
    public List<Long> deadlockedThreadIds = new ArrayList<>();

    public static final class ThreadRecord {
        public long id;
        public String name;
        public boolean daemon;
        public String state;
        public String lockOwner;
        public String lockName;
        public long cpuTimeNanos;
        public long userTimeNanos;
        public List<String> stack = new ArrayList<>();
    }
}
