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

      systems = [ "x86_64-linux" "x86_64-darwin" "aarch64-darwin" ];

      perSystem = { config, self', inputs', pkgs, system, ... }: rec {
        _module.args.pkgs = import inputs.nixpkgs {
          inherit system;

          overlays = [
            (final: prev: {
              dagger = inputs'.dagger.packages.dagger;
              licensei = self'.packages.licensei;
            })
          ];
        };

        devenv.shells = {
          default = {
            languages = {
              go = {
                enable = true;
                package = pkgs.go_1_22;
              };

              python = {
                enable = true;
                package = pkgs.python39;
              };

              javascript = {
                enable = true;
                package = pkgs.nodejs_20;
              };
            };

            pre-commit.hooks = {
              nixpkgs-fmt.enable = true;
              commitizen.enable = true;

              commitizen-branch = {
                enable = true;
                name = "commitizen-branch check";
                description = ''
                  Check whether commit messages on the current HEAD follows committing rules.
                '';
                entry = "${pkgs.commitizen}/bin/cz check --allow-abort --rev-range origin/HEAD..HEAD";
                pass_filenames = false;
                stages = [ "manual" ];
              };
            };

            packages = with pkgs; [
              gnumake
              mage

              # Kafka build dependencies
              rdkafka # https://github.com/confluentinc/confluent-kafka-go#librdkafka
              cyrus_sasl
              pkg-config
              # confluent-platform

              # golangci-lint
              goreleaser
              air

              curl
              jq
              minikube
              kind
              kubectl
              helm-docs

              benthos

              # node
              nodePackages.pnpm

              # python
              poetry

              self'.packages.atlasx

              just
              semver-tool

              dagger
              licensei

              go-migrate
            ];

            env = {
              KUBECONFIG = "${config.devenv.shells.default.env.DEVENV_STATE}/kube/config";
              KIND_CLUSTER_NAME = "openmeter";

              HELM_CACHE_HOME = "${config.devenv.shells.default.env.DEVENV_STATE}/helm/cache";
              HELM_CONFIG_HOME = "${config.devenv.shells.default.env.DEVENV_STATE}/helm/config";
              HELM_DATA_HOME = "${config.devenv.shells.default.env.DEVENV_STATE}/helm/data";

            };

            # https://github.com/cachix/devenv/issues/528#issuecomment-1556108767
            containers = pkgs.lib.mkForce { };
          };

          ci = devenv.shells.default;
        };

        packages = {
          licensei = pkgs.buildGoModule rec {
            pname = "licensei";
            version = "0.8.0";

            src = pkgs.fetchFromGitHub {
              owner = "goph";
              repo = "licensei";
              rev = "v${version}";
              sha256 = "sha256-Pvjmvfk0zkY2uSyLwAtzWNn5hqKImztkf8S6OhX8XoM=";
            };

            vendorHash = "sha256-ZIpZ2tPLHwfWiBywN00lPI1R7u7lseENIiybL3+9xG8=";

            subPackages = [ "cmd/licensei" ];

            ldflags = [
              "-w"
              "-s"
              "-X main.version=v${version}"
            ];
          };

          atlasx = pkgs.stdenv.mkDerivation rec {
            pname = "atlasx";
            version = "0.25.0";
            src = pkgs.fetchurl {
              # License: https://ariga.io/legal/atlas/eula/eula-20240804.pdf
              url = "https://release.ariga.io/atlas/atlas-darwin-arm64-v${version}";
              hash = "sha256-bYJtNDE13UhJWL4ALLKI0sHMZrDS//kFWzguGX63EAo=";
            };

            unpackPhase = ''
              cp $src atlas
            '';

            installPhase = ''
              mkdir -p $out/bin
              cp atlas $out/bin/atlas
            '';

          };
        };
      };
    };
}
