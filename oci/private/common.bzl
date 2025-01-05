""" common utilities """

MEDIA_TYPE_DOCKER_INDEX = "application/vnd.docker.distribution.manifest.list.v2+json"
MEDIA_TYPE_DOCKER_MANIFEST = "application/vnd.docker.distribution.manifest.v2+json"
MEDIA_TYPE_OCI_INDEX = "application/vnd.oci.image.index.v1+json"
MEDIA_TYPE_OCI_MANIFEST = "application/vnd.oci.image.manifest.v1+json"

def get_or_make_descriptor_file(
        ctx,
        *,
        descriptor_provider,  # OCIDescriptor
        outpath = None):  # str | None
    """Returns an oci descriptor file

    Some OCIDescriptor's provider already contain a descrptor file, but others
    only contain starlark variables (strings, bools, dicts, etc.) that are
    the information that would go into a descriptor file.

    This function guarantees you a descriptor file, either by returning the one
    that is already there or by making one on the fly from the information
    contained in the OCIDescriptor provider

    Args:
        ctx: The rule context
        descriptor_provider: An OCIDescriptor that may or may not contain a
            descriptor file
        outpath: Where to declare the new descriptor file if one needs to be
            created. Default is the digest of the descriptor provider
    Returns:
        A descriptor file
    """
    descriptor_file = descriptor_provider.descriptor_file
    if descriptor_file != None:
        return descriptor_file

    if outpath == None:
        outpath = descriptor_provider.digest

    out = ctx.actions.declare_file(outpath)

    obj = {
        _snake_to_camel(k): v
        for k, v in _struct_to_dict(descriptor_provider).items()
        if k in [
            # See: https://github.com/opencontainers/image-spec/blob/main/descriptor.md
            "artifact_type",
            "annotations",
            "data",
            "digest",
            "media_type",
            "platform",
            "size",
            "urls",
        ]
    }

    ctx.actions.write(
        content = json.encode(obj),
        output = out,
        is_executable = False,
    )

    return out

def _snake_to_camel(s):
    components = s.split("_")
    return components[0] + "".join([x.title() for x in components[1:]])

def _struct_to_dict(st):
    return {
        field: getattr(st, field)
        for field in dir(st)
    }
