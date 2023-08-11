load(":pull.bzl", _oci_pull = "oci_pull")
load(":push.bzl", _oci_push = "oci_push")
load(":image.bzl", _oci_image = "oci_image", _oci_image_index = "oci_image_index", _oci_image_layer = "oci_image_layer", _oci_rebase_image = "oci_rebase_image")

oci_pull = _oci_pull
oci_push = _oci_push

oci_image = _oci_image
oci_image_index = _oci_image_index
oci_image_layer = _oci_image_layer
oci_rebase_image = _oci_rebase_image
