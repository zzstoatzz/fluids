const std = @import("std");

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    const mod = b.addModule("fluids", .{
        .root_source_file = b.path("src/root.zig"),
        .target = target,
        .optimize = optimize,
    });

    // unit tests
    const tests = b.addTest(.{ .root_module = mod });
    const test_step = b.step("test", "run unit tests");
    test_step.dependOn(&b.addRunArtifact(tests).step);

    // parity: replay cases recorded by the go oracle (go/cmd/oracle)
    const parity = b.addExecutable(.{
        .name = "parity",
        .root_module = b.createModule(.{
            .root_source_file = b.path("src/parity.zig"),
            .target = target,
            .optimize = optimize,
        }),
    });
    b.installArtifact(parity);
    const run_parity = b.addRunArtifact(parity);
    if (b.args) |args| run_parity.addArgs(args);
    b.step("parity", "compare against go oracle output (pass the json path)").dependOn(&run_parity.step);

    // bench
    const bench = b.addExecutable(.{
        .name = "bench",
        .root_module = b.createModule(.{
            .root_source_file = b.path("src/bench.zig"),
            .target = target,
            .optimize = .ReleaseFast,
        }),
    });
    const run_bench = b.addRunArtifact(bench);
    if (b.args) |args| run_bench.addArgs(args);
    b.step("bench", "time steps natively").dependOn(&run_bench.step);

    // wasm
    const wasm = b.addExecutable(.{
        .name = "fluids",
        .root_module = b.createModule(.{
            .root_source_file = b.path("src/wasm.zig"),
            .target = b.resolveTargetQuery(.{ .cpu_arch = .wasm32, .os_tag = .freestanding }),
            .optimize = .ReleaseFast,
            .strip = true,
        }),
    });
    wasm.entry = .disabled;
    wasm.rdynamic = true;
    const install_wasm = b.addInstallFile(wasm.getEmittedBin(), "../web/fluids.wasm");
    b.step("wasm", "build web/fluids.wasm").dependOn(&install_wasm.step);
}
