{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    devenv.url = "github:cachix/devenv";
    git-hooks.url = "github:cachix/git-hooks.nix";
    pre-commit-hooks.follows = "git-hooks";
    llm-agents = {
      url = "github:numtide/llm-agents.nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-parts.follows = "flake-parts";
    };
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
            inputs.llm-agents.overlays.shared-nixpkgs
          ];
        };

        devenv.shells = {
          default = {
            languages = {
              go = {
                enable = true;
                package = pkgs.go_1_27;
              };

              python = {
                enable = true;
                package = pkgs.python314;
                uv = {
                  enable = true;
                };
              };

              javascript = {
                enable = true;
                package = pkgs.nodejs-slim_26;
                pnpm = {
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

            # Use alternative pre-commit implementations
            git-hooks.package = pkgs.prek;

            packages = with pkgs; [
              git
              gnumake
              mage

              # Kafka build dependencies
              # https://github.com/confluentinc/confluent-kafka-go#librdkafka
              # Check actual version via:
              # $ pkg-config --modversion rdkafka++
              # Getting sha256 hash for git ref:
              # $ nix-shell -p nix-prefetch-git jq --run "nix hash convert sha256:\$(nix-prefetch-git --url https://github.com/confluentinc/librdkafka.git --quiet --rev v2.15.0 | jq -r '.sha256')"
              (rdkafka.overrideAttrs (_: rec {
                src = fetchFromGitHub {
                  owner = "confluentinc";
                  repo = "librdkafka";
                  rev = "v2.15.0";
                  sha256 = "sha256-WW64fwh0xR4lEVwmrv00tP9mo6b49aCNgLLH/P0YS8k=";
                };
              }))

              cyrus_sasl
              pkg-config
              # confluent-platform

              golangci-lint
              goreleaser
              gotestsum
              air

              curl
              jq
              yq-go
              minikube
              kind
              kubectl
              helm-docs
              kubernetes-helm

              benthos

              # We should use a custom light-weight derivation, see this thread https://discourse.nixos.org/t/installing-postgresql-client/948/15
              # Multi-platform support makes this a bit more difficult
              postgresql

              # node: pnpm (from javascript.pnpm) provides `pnpm dlx` as the
              # npx-style runner for these one-off CLIs. The invoked bin runs
              # under the dev shell's `node` (nodejs-slim_26) via its env-node
              # shebang, so no separate corepack/Node runtime is needed.
              # We can consider adding a pkgs.buildNpmPackage for spectral-cli if build takes a lot of time, but for now
              # this is a quick fix to get it working.
              (writeShellScriptBin "spectral" ''
                exec ${pkgs.pnpm}/bin/pnpm dlx @stoplight/spectral-cli@6.16.0 "$@"
              '')
              self'.packages.codegraph

              # python
              poetry

              self'.packages.atlasx

              just
              semver-tool

              go-migrate

              sqlc
            ];

            env = {
              GOCACHE = "${config.devenv.shells.default.env.DEVENV_STATE}/go/build-cache";
              KUBECONFIG = "${config.devenv.shells.default.env.DEVENV_STATE}/kube/config";
              KIND_CLUSTER_NAME = "openmeter";

              HELM_CACHE_HOME = "${config.devenv.shells.default.env.DEVENV_STATE}/helm/cache";
              HELM_CONFIG_HOME = "${config.devenv.shells.default.env.DEVENV_STATE}/helm/config";
              HELM_DATA_HOME = "${config.devenv.shells.default.env.DEVENV_STATE}/helm/data";
            };

            enterShell = ''
              # Share downloaded modules across worktrees; keep GOPATH and build artifacts local.
              export GOMODCACHE="$HOME/go/pkg/mod"

              ${lib.optionalString pkgs.stdenv.hostPlatform.isDarwin ''
                # Workaround for XCBUILD.XCRUN cosmetic issue due to incompatible plists (see https://github.com/NixOS/nixpkgs/issues/376958)
                # 1) Filter out the buggy Nix version of xcbuild/xcrun from PATH
                export PATH=$(echo "$PATH" | tr ':' '\n' | grep -v "xcbuild" | tr '\n' ':')

                # 2) Force the system to use the Apple SDK path if needed
                unset DEVELOPER_DIR
                # End of workaround
              ''}

              # Keep GitHub-hosted Node jobs aligned with the Nix shell.
              node -v > .nvmrc
            '';

            # https://github.com/cachix/devenv/issues/528#issuecomment-1556108767
            containers = pkgs.lib.mkForce { };
          };

          ci = devenv.shells.default;
        };

        packages = {
          # CodeGraph 1.5.0 aborts while indexing on macOS with Node 24.
          codegraph = pkgs.llm-agents.codegraph.override {
            buildNpmPackage = pkgs.buildNpmPackage.override { nodejs = pkgs.nodejs_22; };
          };
        } // import ./custom-packages.nix { inherit pkgs; };
      };
    };
}
