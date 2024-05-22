load(":pull.bzl", _oci_pull = "oci_pull")
load(":push.bzl", _oci_push = "oci_push")
load(":image.bzl", _oci_image = "oci_image", _oci_image_index = "oci_image_index", _oci_image_layer = "oci_image_layer")
load(":oci_image_layout.bzl", _oci_image_layout = "oci_image_layout")

oci_pull = _oci_pull
oci_push = _oci_push

oci_image = _oci_image
oci_image_index = _oci_image_index
oci_image_layer = _oci_image_layer
oci_image_layout = _oci_image_layout
