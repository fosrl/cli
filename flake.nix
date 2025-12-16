{
  description = "pangolin-cli - a VPN client for pangolin";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = {nixpkgs, ...}: let
    supportedSystems = [
      "x86_64-linux"
      "aarch64-linux"
      "x86_64-darwin"
      "aarch64-darwin"
    ];
    forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    pkgsFor = system: nixpkgs.legacyPackages.${system};
  in {
    devShells = forAllSystems (
      system: let
        pkgs = pkgsFor system;

        inherit
          (pkgs)
          go
          golangci-lint
          ;
      in {
        default = pkgs.mkShell {
          buildInputs = [
            go
            golangci-lint
          ];
        };
      }
    );
  };
}
