package ociutil

// TODO use upstream defs when a new release is cut
// https://github.com/opencontainers/image-spec/commit/71ccc68078c473544315863eabb2f95140f7e1bf#diff-05a9698dc79be9f08ba5b6fbbaa6bb013a61c3b2db9b5cd1aa570677f7065c0c
var (
	// AnnotationBaseImageDigest is the annotation key for the digest of the image's base image.
	AnnotationBaseImageDigest = "org.opencontainers.image.base.digest"

	// AnnotationBaseImageName is the annotation key for the image reference of the image's base image.
	AnnotationBaseImageName = "org.opencontainers.image.base.name"
)
