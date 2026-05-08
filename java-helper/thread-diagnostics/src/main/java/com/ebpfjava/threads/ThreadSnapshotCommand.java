package com.ebpfjava.threads;

import java.lang.management.ManagementFactory;
import java.lang.management.ThreadInfo;
import java.lang.management.ThreadMXBean;

public final class ThreadSnapshotCommand {
    private final ThreadMXBean threadBean;
    private final int maxDepth;

    public ThreadSnapshotCommand(ThreadMXBean threadBean, int maxDepth) {
        this.threadBean = threadBean;
        this.maxDepth = Math.max(1, maxDepth);
    }

    public static ThreadSnapshotPayload captureDefault(int maxDepth) {
        return new ThreadSnapshotCommand(ManagementFactory.getThreadMXBean(), maxDepth).capture();
    }

    public ThreadSnapshotPayload capture() {
        ThreadSnapshotPayload payload = new ThreadSnapshotPayload();
        payload.capturedAtMillis = System.currentTimeMillis();
        payload.threadCpuTimeSupported = threadBean.isThreadCpuTimeSupported();
        payload.contentionMonitoringEnabled = threadBean.isThreadContentionMonitoringEnabled();
        ThreadInfo[] infos = threadBean.dumpAllThreads(true, true);
        for (ThreadInfo info : infos) {
            if (info == null) {
                continue;
            }
            ThreadSnapshotPayload.ThreadRecord record = new ThreadSnapshotPayload.ThreadRecord();
            record.id = info.getThreadId();
            record.name = info.getThreadName();
            record.state = info.getThreadState().name();
            record.lockOwner = info.getLockOwnerName();
            record.lockName = info.getLockName();
            if (payload.threadCpuTimeSupported) {
                record.cpuTimeNanos = threadBean.getThreadCpuTime(info.getThreadId());
                record.userTimeNanos = threadBean.getThreadUserTime(info.getThreadId());
            }
            StackTraceElement[] stack = info.getStackTrace();
            for (int i = 0; i < stack.length && i < maxDepth; i++) {
                record.stack.add(stack[i].toString());
            }
            payload.threads.add(record);
        }
        long[] deadlocked = threadBean.findDeadlockedThreads();
        if (deadlocked != null) {
            for (long id : deadlocked) {
                payload.deadlockedThreadIds.add(id);
            }
        }
        return payload;
    }
}
