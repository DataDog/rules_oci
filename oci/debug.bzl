DebugInfo = provider(fields = ['debug'])

def _debug_flag_impl(ctx):
    return [DebugInfo(debug = ctx.build_setting_value)]

debug_flag = rule(
    implementation = _debug_flag_impl,
    build_setting = config.bool(flag = True)
)
