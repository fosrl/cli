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
        pangolin-cli = pkgs.buildGoModule {
          pname = "pangolin-cli";
          version = "0.1.0";
          src = ./.;

          vendorHash = "sha256-C6Kpug1pcH2uEK22Zr6IQsfUiSj+wSVc8vOqLgWg0hk=";

          ldflags = [
            "-s"
            "-w"
          ];

          meta = with pkgs.lib; {
            description = "A VPN client for pangolin";
            homepage = "https://github.com/fosrl/cli";
            license = licenses.unfree;
            maintainers = [];
            mainProgram = "pangolin-cli";
          };
        };

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