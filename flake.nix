{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    devenv.url = "github:cachix/devenv";
    dagger.url = "github:dagger/nix";
    dagger.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.devenv.flakeModule
      ];

      systems = [ "x86_64-linux" "x86_64-darwin" "aarch64-darwin" "aarch64-linux" ];

      perSystem = { config, self', inputs', pkgs, lib, system, ... }: rec {
        # FIX: Removed the 'system' argument to fix evaluation warnings
        _module.args.pkgs = import inputs.nixpkgs {
          localSystem = system; # Use localSystem instead of system

          overlays = [
            (final: prev: {
              dagger = inputs'.dagger.packages.dagger;
              atlasx = self'.packages.atlasx;
            })
          ];
        };

        devenv.shells = {
          default = {
            languages = {
              go = {
                enable = true;
                package = pkgs.go_1_25;
              };

              python = {
                enable = true;
                package = pkgs.python314;
                uv.enable = true;
              };

              javascript = {
                enable = true;
                package = pkgs.nodejs_24;
                corepack.enable = true;
              };
            };

            git-hooks.hooks = {
              nixpkgs-fmt.enable = true;
              commitizen.enable = true;

              commitizen-branch = {
                enable = true;
                name = "commitizen-branch check";
                description = "Check whether commit messages on the current HEAD follows committing rules.";
                entry = "${pkgs.commitizen}/bin/cz check --allow-abort --rev-range origin/HEAD..HEAD";
                pass_filenames = false;
                stages = [ "manual" ];
              };
            };

            packages = with pkgs; [
              # --- FIX: ADDED PRE-COMMIT HERE ---
              pre-commit
              # ----------------------------------

              gnumake
              mage
              (rdkafka.overrideAttrs (_: rec {
                src = fetchFromGitHub {
                  owner = "confluentinc";
                  repo = "librdkafka";
                  rev = "v2.13.0";
                  sha256 = "sha256-gxZ20qpG3iXwY21fY2lvafWudcnsqN6hOml1UR9fPKQ=";
                };
              }))
              cyrus_sasl
              pkg-config
              golangci-lint
              goreleaser
              air
              curl
              jq
              minikube
              kind
              kubectl
              helm-docs
              kubernetes-helm
              benthos
              postgresql
              corepack_24
              (writeShellScriptBin "spectral" ''
                exec ${pkgs.nodejs_24}/bin/npx -y @stoplight/spectral-cli@6.13.1 "$@"
              '')
              poetry
              atlasx
              just
              semver-tool
              dagger
              go-migrate
              sqlc
            ];

            env = {
              KUBECONFIG = "${config.devenv.shells.default.env.DEVENV_STATE}/kube/config";
              KIND_CLUSTER_NAME = "openmeter";
              HELM_CACHE_HOME = "${config.devenv.shells.default.env.DEVENV_STATE}/helm/cache";
              HELM_CONFIG_HOME = "${config.devenv.shells.default.env.DEVENV_STATE}/helm/config";
              HELM_DATA_HOME = "${config.devenv.shells.default.env.DEVENV_STATE}/helm/data";
            };

            enterShell = lib.optionalString pkgs.stdenv.isDarwin ''
              export PATH=$(echo "$PATH" | tr ':' '\n' | grep -v "xcbuild" | tr '\n' ':')
              unset DEVELOPER_DIR
            '';

            containers = pkgs.lib.mkForce { };
          };

          ci = devenv.shells.default;

          dagger = {
            languages.go = devenv.shells.default.languages.go;
            packages = with pkgs; [ gnumake git atlasx ];
            containers = devenv.shells.default.containers;
          };
        };

        packages.atlasx =
          let
            systemMappings = {
              x86_64-linux = "linux-amd64";
              x86_64-darwin = "darwin-amd64";
              aarch64-darwin = "darwin-arm64";
              aarch64-linux = "linux-arm64";
            };
            hashMappings = {
              x86_64-linux = "sha256-2IquGGpV5Yk8MY87Ecg4ozcq302sHi/TvH0rVZRMV5c=";
              x86_64-darwin = "sha256-yMvFQ32wVAXpzXEN+hC8nTkr+2eqoWBhT92JqXBUusQ=";
              aarch64-darwin = "sha256-mP7mg4RyqdL5D5FFNEna6aWs/cEsNq/vrmdiX78/EP0=";
              aarch64-linux = "sha256-u4oioIzNmmy5PwoWIFt7vrBn3X/sH2AifGh9jek9YIg=";
            };
          in
          pkgs.stdenv.mkDerivation rec {
            pname = "atlasx";
            version = "0.36.0";
            src = pkgs.fetchurl {
              url = "https://release.ariga.io/atlas/atlas-${systemMappings."${system}"}-v${version}";
              hash = hashMappings."${system}";
            };
            unpackPhase = "cp $src atlas";
            installPhase = ''
              mkdir -p $out/bin
              cp atlas $out/bin/atlas
              chmod +x $out/bin/atlas
            '';
          };
      };
    };
}
