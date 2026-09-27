{
  description = "FoodList development environment";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      devShells = forAllSystems (system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go_1_27
              nodejs_24
              gnumake
              git
              air
              golangci-lint
              pkg-config
              libheif
            ];

            shellHook = ''
              echo "FoodList development environment is ready."
              echo "Run 'make build', 'make test', or 'make dev'."
            '';
          };
        });
    };
}
