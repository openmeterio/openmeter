{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    nixpkgs-node.url = "github:NixOS/nixpkgs/04a1d3aa7f0c1ef4fd84974b757ba4c279747642";
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

      perSystem = { config, self', inputs', pkgs, lib, system, ... }:
        let
          # The Nixpkgs Go package can retain a build-time loader that is not represented in
          # its store identity. Wrap cmd/link with the active loader as an explicit argument
          # and derivation input so every internal link, including Nix checks, uses it.
          goPackage =
            if pkgs.stdenv.hostPlatform.isLinux then
              pkgs.symlinkJoin
                {
                  name = "${pkgs.go_1_27.name}-openmeter";
                  paths = [ pkgs.go_1_27 ];
                  inherit (pkgs.go_1_27) CGO_ENABLED GOARCH GOOS meta passthru version;
                  nativeBuildInputs = [ pkgs.makeWrapper ];
                  postBuild = ''
                    linkTool="share/go/pkg/tool/${pkgs.go_1_27.GOOS}_${pkgs.go_1_27.GOARCH}/link"
                    rm "$out/$linkTool"
                    makeWrapper "${pkgs.go_1_27}/$linkTool" "$out/$linkTool" \
                      --add-flags "-I=${pkgs.stdenv.cc.bintools.dynamicLinker}"

                    wrapProgram "$out/bin/go" \
                      --set-default GO_LDSO ${pkgs.stdenv.cc.bintools.dynamicLinker}
                  '';
                }
            else
              pkgs.go_1_27;

          goShell = shellName: {
            languages.go = {
              enable = true;
              package = goPackage;

              delve.package = pkgs.delve.overrideAttrs (old: {
                # Delve runs these generator checks only with the latest Go release. They invoke
                # `go run ...@latest`, which cannot query modules while buildGoModule uses -mod=vendor.
                # Keep the remaining Delve tests enabled until upstream packaging skips these checks.
                checkFlags = (old.checkFlags or [ ]) ++ [
                  "-skip=TestGeneratedDoc|TestTypecheckRPC"
                ];
              });
            };

            packages = with pkgs; [
              git
              gnumake
              gnupatch
            ];

            env = {
              # Go's action cache does not include Nix store identities in its linker key. Include
              # both the complete Go output and runtime-loader store basenames so neither a Go
              # rebuild nor a stdenv/glibc update can reuse link results from another generation.
              GOCACHE =
                let
                  goToolchainID = builtins.baseNameOf (toString goPackage);
                  runtimeLinkerID =
                    if pkgs.stdenv.hostPlatform.isLinux then
                      builtins.elemAt (builtins.match "/nix/store/([^/]+)/.*" pkgs.stdenv.cc.bintools.dynamicLinker) 0
                    else
                      "native";
                in
                "${config.devenv.shells.${shellName}.env.DEVENV_STATE}/go/build-cache/${goToolchainID}/${runtimeLinkerID}";
            } // lib.optionalAttrs pkgs.stdenv.hostPlatform.isLinux {
              # Keep GO_LDSO visible for patched Go linkers and diagnostics. The wrapped cmd/link
              # above also passes -I explicitly because the cached Nix binary can retain its stale
              # build-time default even when this variable is set.
              GO_LDSO = pkgs.stdenv.cc.bintools.dynamicLinker;
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
            '';

            # https://github.com/cachix/devenv/issues/528#issuecomment-1556108767
            containers = pkgs.lib.mkForce { };
          };

          apiShell = shellName:
            let
              base = goShell shellName;
            in
            lib.recursiveUpdate base {
              languages.javascript = {
                enable = true;
                package = inputs'.nixpkgs-node.legacyPackages.nodejs-slim_26;
                pnpm.enable = true;
              };

              packages = base.packages ++ [ pkgs.yq-go ];

              enterShell = base.enterShell + ''
                # Keep GitHub-hosted Node jobs aligned with the Nix shell.
                node -v > .nvmrc
              '';
            };

          fullShell = shellName:
            let
              base = apiShell shellName;
            in
            lib.recursiveUpdate base {
              languages.python = {
                enable = true;
                package = pkgs.python314;
                uv.enable = true;
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

              packages = base.packages ++ (with pkgs; [
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
              ]);

              env = {
                KUBECONFIG = "${config.devenv.shells.${shellName}.env.DEVENV_STATE}/kube/config";
                KIND_CLUSTER_NAME = "openmeter";

                HELM_CACHE_HOME = "${config.devenv.shells.${shellName}.env.DEVENV_STATE}/helm/cache";
                HELM_CONFIG_HOME = "${config.devenv.shells.${shellName}.env.DEVENV_STATE}/helm/config";
                HELM_DATA_HOME = "${config.devenv.shells.${shellName}.env.DEVENV_STATE}/helm/data";
              };
            };
        in
        rec {
          _module.args.pkgs = import inputs.nixpkgs {
            inherit system;

            overlays = [
              inputs.llm-agents.overlays.shared-nixpkgs
            ];
          };

          devenv.shells = {
            go = goShell "go";
            api = apiShell "api";
            default = fullShell "default";
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
