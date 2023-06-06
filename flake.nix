{
  description = "OpenMeter streamlines real-time metering data collection and accurate aggregation";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;

          overlays = [
            (final: prev: {
              rdkafka = prev.rdkafka.overrideAttrs (f: p: rec {
                version = "2.1.1";

                src = prev.fetchFromGitHub {
                  owner = "confluentinc";
                  repo = "librdkafka";
                  rev = "v${version}";
                  sha256 = "sha256-MwPRnD/S8o1gG6RWq2tKxqdpGum4FB5K8bHPAvlKW10=";
                };
              });
            })
          ];
        };
      in
      {
        devShells = {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              git
              gnumake

              go_1_20
              rdkafka # https://github.com/confluentinc/confluent-kafka-go#librdkafka
              cyrus_sasl
              pkg-config
              golangci-lint
              goreleaser
              mage
              air
              oapi-codegen

              curl
              jq
              minikube
            ];
          };

          ci = pkgs.mkShell {
            buildInputs = with pkgs; [
              git
              gnumake

              go_1_20
              rdkafka # https://github.com/confluentinc/confluent-kafka-go#librdkafka
              cyrus_sasl
              pkg-config
              golangci-lint
              goreleaser
              mage
              oapi-codegen
            ];
          };
        };
      }
    );
}
