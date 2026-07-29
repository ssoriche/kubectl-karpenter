{
  description = "kubectl-karpenter: Show Karpenter NodePool resource utilization";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems =
        f: nixpkgs.lib.genAttrs supportedSystems (system: f system nixpkgs.legacyPackages.${system});
    in
    {
      # Overlay that adds `kubectl-karpenter` to a nixpkgs instance,
      # for consumers who prefer composing it into their own package set.
      overlays.default = final: _prev: {
        kubectl-karpenter = final.callPackage ./nix/package.nix { };
      };

      # `nix build`, `nix build .#kubectl-karpenter`, and the input for
      # the NixOS / home-manager modules below.
      packages = forAllSystems (
        _system: pkgs: rec {
          kubectl-karpenter = pkgs.callPackage ./nix/package.nix { };
          default = kubectl-karpenter;
        }
      );

      # `nix run` -> `kubectl-karpenter` (also usable as `kubectl karpenter`).
      apps = forAllSystems (
        system: _pkgs: rec {
          kubectl-karpenter = {
            type = "app";
            program = nixpkgs.lib.getExe self.packages.${system}.kubectl-karpenter;
            meta.description = "Show Karpenter NodePool resource utilization";
          };
          default = kubectl-karpenter;
        }
      );

      # Matches the flox dev environment: Go 1.26 plus the release tooling.
      devShells = forAllSystems (
        _system: pkgs: {
          default = pkgs.mkShellNoCC {
            packages = [
              pkgs.go_1_26
              pkgs.golangci-lint
              pkgs.goreleaser
              pkgs.kubectl
            ];
          };
        }
      );

      formatter = forAllSystems (_system: pkgs: pkgs.nixfmt);

      # Declarative installation of the krew/kubectl plugin on NixOS or via
      # home-manager. Enabling either puts `kubectl-karpenter` on PATH,
      # which is all kubectl needs to expose `kubectl karpenter`.
      nixosModules.default = import ./nix/nixos-module.nix self;
      homeManagerModules.default = import ./nix/home-manager-module.nix self;
    };
}
