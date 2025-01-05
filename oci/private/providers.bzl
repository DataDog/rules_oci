""" providers """

DebugInfo = provider(
    "DebugInfo",
    fields = ["debug"],
)

PlatformsInfo = provider(
    doc = "Information about the platforms of each manifest in an OCI image",
    fields = {
        "platforms": "a json file containing information about the platforms of each manifest in an OCI image",
    },
)
