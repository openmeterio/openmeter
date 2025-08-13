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
        _module.args.pkgs = import inputs.nixpkgs {
          inherit system;

          overlays = [
            (final: prev: {
              dagger = inputs'.dagger.packages.dagger;
              atlasx = self'.packages.atlasx;
            })
          ];
        };

        devenv.shells = {
          default =
            let
              corepack031 = pkgs.corepack.overrideAttrs (_: {
                version = "0.31.0";
                src = pkgs.fetchurl {
                  url = "https://registry.npmjs.org/corepack/-/corepack-0.31.0.tgz";
                  sha256 = "sha256-fGMkPPHeJV2BKJ+LRzzsVIwaltF2DAw2ltb9M2pj6xc=";
                };
              });
            in
            {
              languages = {
                go = {
                  enable = true;
                  package = pkgs.go_1_24;
                };

                python = {
                  enable = true;
                  package = pkgs.python312;
                };

                javascript = {
                  enable = true;
                  package = pkgs.nodejs_22;
                  corepack = {
                    enable = true;
                  };
                };
              };

              git-hooks.hooks = {
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
                # https://github.com/confluentinc/confluent-kafka-go#librdkafka
                # Check actual version via:
                # $ pkg-config --modversion rdkafka++
                # Getting sha256 hash for git ref:
                # $ nix-shell -p nix-prefetch-git jq --run "nix hash convert sha256:\$(nix-prefetch-git --url https://github.com/confluentinc/librdkafka.git --quiet --rev v2.11.0 | jq -r '.sha256')"
                (rdkafka.overrideAttrs (_: rec {
                  version = "2.11.0";
                  src = fetchFromGitHub {
                    owner = "confluentinc";
                    repo = "librdkafka";
                    rev = "v${version}";
                    sha256 = "sha256-37lCQ+CFeTRQwL6FCl79RSGw+nRKr0DeuXob9CjiVnk=";
                  };
                }))

                cyrus_sasl
                pkg-config
                # confluent-platform

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

                # We should use a custom light-weight derivation, see this thread https://discourse.nixos.org/t/installing-postgresql-client/948/15
                # Multi-platform support makes this a bit more difficult
                postgresql

                # node
                nodePackages.pnpm
                corepack031
                # We can consider adding a pkgs.buildNpmPackage for spectral-cli if build takes a lot of time, but for now
                # this is a quick fix to get it working.
                (writeShellScriptBin "spectral" ''
                  exec ${pkgs.nodejs_22}/bin/npx -y @stoplight/spectral-cli@6.13.1 "$@"
                '')

                # python
                poetry

                atlasx

                just
                semver-tool

                dagger

                go-migrate

                sqlc
              ];


              enterShell = ''
                # Put Corepack 0.31.0 first on PATH
                export PATH="${corepack031}/bin:$PATH"

                # Disable download prompt for corepack
                export COREPACK_ENABLE_DOWNLOAD_PROMPT=0
              '';

              env = {
                KUBECONFIG = "${config.devenv.shells.default.env.DEVENV_STATE}/kube/config";
                KIND_CLUSTER_NAME = "openmeter";

                HELM_CACHE_HOME = "${config.devenv.shells.default.env.DEVENV_STATE}/helm/cache";
                HELM_CONFIG_HOME = "${config.devenv.shells.default.env.DEVENV_STATE}/helm/config";
                HELM_DATA_HOME = "${config.devenv.shells.default.env.DEVENV_STATE}/helm/data";

                PATH = lib.mkBefore "${
                (pkgs.corepack.overrideAttrs (_: {
                  version = "0.31.0";
                  src = pkgs.fetchurl {
                    url = "https://registry.npmjs.org/corepack/-/corepack-0.31.0.tgz";
                    sha256 = "sha256-fGMkPPHeJV2BKJ+LRzzsVIwaltF2DAw2ltb9M2pj6xc=";
                  };
                }))
              }/bin";
              };

              # https://github.com/cachix/devenv/issues/528#issuecomment-1556108767
              containers = pkgs.lib.mkForce { };
            };

          ci = devenv.shells.default;

          # Lighteweight target to use inside dagger
          dagger = {
            languages = {
              go = devenv.shells.default.languages.go;
            };
            packages = with pkgs; [
              gnumake
              git
              atlasx
            ];
            containers = devenv.shells.default.containers;
          };
        };

        packages = {
          atlasx =
            let
              systemMappings = {
                x86_64-linux = "linux-amd64";
                x86_64-darwin = "darwin-amd64";
                aarch64-darwin = "darwin-arm64";
                aarch64-linux = "linux-arm64";
              };
              # nix hash convert --hash-algo sha256 --to sri SHA256SUM
              hashMappings = {
                # nix hash convert --hash-algo sha256 --to sri "$(curl -sfL 'https://release.ariga.io/atlas/atlas-linux-amd64-v'"${VERSION}"'.sha256')"
                x86_64-linux = "sha256-YzakYgbSjb3Zvzu7v8AgD/TH9Gwik6WyOLrK7qjX6N4=";
                # nix hash convert --hash-algo sha256 --to sri "$(curl -sfL 'https://release.ariga.io/atlas/atlas-darwin-amd64-v'"${VERSION}"'.sha256')"
                x86_64-darwin = "sha256-MMfjuxZygcNoFFRoFDsSihHn+l6CD/J2gaResqFnrvI=";
                # nix hash convert --hash-algo sha256 --to sri "$(curl -sfL 'https://release.ariga.io/atlas/atlas-darwin-arm64-v'"${VERSION}"'.sha256')"
                aarch64-darwin = "sha256-4a7apzaozkoF9SZnve18wQGXBdOAwdqB27TSrSS6n0Y=";
                # nix hash convert --hash-algo sha256 --to sri "$(curl -sfL 'https://release.ariga.io/atlas/atlas-linux-arm64-v'"${VERSION}"'.sha256')"
                aarch64-linux = "sha256-Xhq4s93xFzFBQSVtiKoTGYMldA/WFIE0icFahNp6wXU=";
              };
            in
            pkgs.stdenv.mkDerivation rec {
              pname = "atlasx";
              version = "0.32.1";

              src = pkgs.fetchurl {
                # License: https://ariga.io/legal/atlas/eula/eula-20240804.pdf
                url = "https://release.ariga.io/atlas/atlas-${systemMappings."${system}"}-v${version}";
                hash = hashMappings."${system}";
              };

              unpackPhase = ''
                cp $src atlas
              '';

              installPhase = ''
                mkdir -p $out/bin
                cp atlas $out/bin/atlas
                chmod +x $out/bin/atlas
              '';

            };
        };
      };
    };
}
