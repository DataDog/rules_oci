"""with_platform"""

def _transition_platform_impl(_settings, attr):
    if attr.os == "darwin":
        return {}

    platform = "@zig_sdk//platform:{os}_{arch}".format(arch = attr.arch, os = attr.os)
    return {
        "//command_line_option:platforms": platform,
    }

transition_platform = transition(
    implementation = _transition_platform_impl,
    inputs = [],
    outputs = [
        "//command_line_option:platforms",
    ],
)

def _with_platform_impl(ctx):
    src = ctx.attr.src[0]

    original_executable = src[DefaultInfo].files_to_run.executable
    if original_executable == None:
        return [
            src[DefaultInfo],
        ]

    # Executables need to be handled specially, as we are not allowed to just
    # forward the executable into a new DefaultInfo provider. The executable
    # returned from this rule must be created by an action inside this rule;
    # so we create an action that copies the old executable to a new executable

    new_executable = ctx.actions.declare_file(ctx.label.name)

    ctx.actions.run_shell(
        outputs = [new_executable],
        inputs = [original_executable],
        command = "cp {input} {output}".format(
            input = original_executable.path,
            output = new_executable.path,
        ),
    )

    runfiles = ctx.runfiles([new_executable])
    runfiles = runfiles.merge(src[DefaultInfo].default_runfiles)

    return [
        DefaultInfo(
            files = depset([new_executable]),
            runfiles = runfiles,
            executable = new_executable,
        ),
    ]

with_platform = rule(
    implementation = _with_platform_impl,
    attrs = {
        "arch": attr.string(mandatory = True),
        "os": attr.string(mandatory = True),
        "src": attr.label(
            cfg = transition_platform,
            allow_single_file = True,
            mandatory = True,
            providers = [DefaultInfo],
        ),
    },
)
