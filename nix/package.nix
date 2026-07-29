{
  lib,
  buildGoModule,
  go_1_26,
  versionCheckHook,
}:

let
  version = "0.1.0";
in
# Pin the Go toolchain to 1.26 to satisfy the `go 1.26` directive in go.mod;
# a sandboxed Nix build cannot download a newer toolchain on demand.
(buildGoModule.override { go = go_1_26; }) {
  pname = "kubectl-karpenter";
  inherit version;

  # Only the files needed to build the binary, so unrelated changes
  # (docs, CI config, the flox env) don't invalidate the build.
  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../go.mod
      ../go.sum
      ../cmd
      ../internal
    ];
  };

  vendorHash = "sha256-JyrEXuI9B1dPhhAtzpiz4s7+T4yUaRBH74PKky9Q4JI=";

  subPackages = [ "cmd/kubectl-karpenter" ];

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-w"
    "-X main.version=${version}"
  ];

  # Sanity-check that the built binary runs and reports its version.
  nativeBuildInputs = [ versionCheckHook ];
  doInstallCheck = true;

  meta = {
    description = "kubectl/krew plugin that shows Karpenter NodePool resource utilization";
    longDescription = ''
      Displays a summary of Karpenter NodePools with node counts and CPU/memory
      request utilization shown as ASCII bar charts.

      Because kubectl discovers `kubectl-*` executables on PATH as plugins,
      installing this package makes `kubectl karpenter` available.
    '';
    homepage = "https://github.com/ssoriche/kubectl-karpenter";
    license = lib.licenses.mit;
    mainProgram = "kubectl-karpenter";
    maintainers = [ ];
  };
}
