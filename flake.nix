{
  description = "OpenMeter streamlines real-time metering data collection and accurate aggregation";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    devenv.url = "github:cachix/devenv";
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.devenv.flakeModule
      ];

      systems = [ "x86_64-linux" "x86_64-darwin" "aarch64-darwin" ];

      perSystem = { config, self', inputs', pkgs, system, ... }: rec {
        devenv.shells = {
          default = {
            languages = {
              go.enable = true;
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
              dagger
              mage

              # Kafka build dependencies
              rdkafka # https://github.com/confluentinc/confluent-kafka-go#librdkafka
              cyrus_sasl
              pkg-config
              # confluent-platform

              golangci-lint
              goreleaser
              air

              curl
              jq
              minikube
            ] ++ [
              self'.packages.licensei
            ];

            scripts = {
              versions.exec = ''
                go version
                golangci-lint version
              '';
            };

            enterShell = ''
              versions
            '';

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

            vendorSha256 = "sha256-ZIpZ2tPLHwfWiBywN00lPI1R7u7lseENIiybL3+9xG8=";

            subPackages = [ "cmd/licensei" ];

            ldflags = [
              "-w"
              "-s"
              "-X main.version=v${version}"
            ];
          };
        };
      };
    };
}
