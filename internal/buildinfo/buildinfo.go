package buildinfo

// Version is injected at build time via -ldflags "-X .../buildinfo.Version=x".
var Version = "dev"
