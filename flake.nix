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
    packages = forAllSystems (
      system: let
        pkgs = pkgsFor system;
      in rec {
        pangolin-cli = pkgs.buildGoModule (finalAttrs: {
          pname = "pangolin-cli";
          version = "0.20.0";
          src = ./.;

          vendorHash = "sha256-c3gwh1w5kO45vqhAG44yGiMRkODUKwnbM/grJmyFUME=";

          ldflags = [
            "-s"
            "-w"
            "-X github.com/fosrl/cli/internal/version.Version=${finalAttrs.version}"
          ];

          meta = with pkgs.lib; {
            description = "A VPN client for pangolin";
            homepage = "https://github.com/fosrl/cli";
            license = licenses.unfree;
            maintainers = [];
            mainProgram = "pangolin-cli";
          };
        });

        default = pangolin-cli;
      }
    );

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
