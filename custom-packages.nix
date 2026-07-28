{ pkgs }:

let
  inherit (pkgs) fetchurl stdenv;

  systemMappings = {
    x86_64-linux = "linux-amd64";
    x86_64-darwin = "darwin-amd64";
    aarch64-darwin = "darwin-arm64";
    aarch64-linux = "linux-arm64";
  };

  # nix hash convert --hash-algo sha256 --to sri SHA256SUM
  hashMappings = {
    # nix hash convert --hash-algo sha256 --to sri "$(curl -sfL 'https://release.ariga.io/atlas/atlas-linux-amd64-v'"${VERSION}"'.sha256')"
    x86_64-linux = "sha256-2IquGGpV5Yk8MY87Ecg4ozcq302sHi/TvH0rVZRMV5c=";
    # nix hash convert --hash-algo sha256 --to sri "$(curl -sfL 'https://release.ariga.io/atlas/atlas-darwin-amd64-v'"${VERSION}"'.sha256')"
    x86_64-darwin = "sha256-yMvFQ32wVAXpzXEN+hC8nTkr+2eqoWBhT92JqXBUusQ=";
    # nix hash convert --hash-algo sha256 --to sri "$(curl -sfL 'https://release.ariga.io/atlas/atlas-darwin-arm64-v'"${VERSION}"'.sha256')"
    aarch64-darwin = "sha256-mP7mg4RyqdL5D5FFNEna6aWs/cEsNq/vrmdiX78/EP0=";
    # nix hash convert --hash-algo sha256 --to sri "$(curl -sfL 'https://release.ariga.io/atlas/atlas-linux-arm64-v'"${VERSION}"'.sha256')"
    aarch64-linux = "sha256-u4oioIzNmmy5PwoWIFt7vrBn3X/sH2AifGh9jek9YIg=";
  };
  atlasx = stdenv.mkDerivation rec {
    pname = "atlasx";
    version = "0.36.0";

    src = fetchurl {
      # License: https://ariga.io/legal/atlas/eula/eula-20240804.pdf
      url = "https://release.ariga.io/atlas/atlas-${systemMappings."${stdenv.hostPlatform.system}"}-v${version}";
      hash = hashMappings."${stdenv.hostPlatform.system}";
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
in
{
  inherit atlasx;
}
