package com.ebpfjava.threads;

import java.lang.instrument.Instrumentation;

public final class ThreadDiagnosticsAgent {
    private ThreadDiagnosticsAgent() {
    }

    public static void premain(String args, Instrumentation instrumentation) {
        agentmain(args, instrumentation);
    }

    public static void agentmain(String args, Instrumentation instrumentation) {
        int depth = 128;
        if (args != null && !args.isBlank()) {
            depth = Integer.parseInt(args.trim());
        }
        ThreadSnapshotCommand.captureDefault(depth);
    }
}
