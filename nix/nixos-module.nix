self:
{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.programs.kubectl-karpenter;
in
{
  options.programs.kubectl-karpenter = {
    enable = lib.mkEnableOption ''
      kubectl-karpenter, a kubectl/krew plugin that shows Karpenter NodePool resource utilization. Installing
      it puts the `kubectl-karpenter` binary on PATH, so `kubectl karpenter` becomes available system-wide'';

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.kubectl-karpenter;
      defaultText = lib.literalExpression "kubectl-karpenter.packages.\${system}.kubectl-karpenter";
      description = "The kubectl-karpenter package to install.";
    };
  };

  config = lib.mkIf cfg.enable {
    environment.systemPackages = [ cfg.package ];
  };
}
