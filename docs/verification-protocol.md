# Verification protocol

The pull request workflow checks Go formatting, compiles the command, runs the Go tests with an explicit non-cache test mode, runs vet, and invokes the canonical corpus twice with the same CI runtime receipt. It compares all seven files byte-for-byte and checks that the checkout has no repository write.

The post-merge workflow uploads the same seven files as one Actions artifact. The release record is completed only after the tag target, immutable release setting, release asset digest, Actions run/job, and uploaded artifact are independently read back from GitHub.
