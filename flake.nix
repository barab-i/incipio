{
  description = "A simple application launcher";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = nixpkgs.legacyPackages.${system};
      filteredSrc = pkgs.lib.sources.cleanSourceWith {
        src = ./.;
        filter = path: type:
          let
            baseName = baseNameOf (toString path);
          in
          !(pkgs.lib.strings.hasSuffix "/examples" (toString path)) &&
          baseName != ".git";
      };
    in {
      packages.default = pkgs.buildGoModule {
        pname = "incipio";
        version = "0.1.0";

        src = filteredSrc;

        vendorHash = "sha256-JONIyyzfLYTQ9fw3or7ljSId7fpwdkJ+dwZbVmoVDAc=";

        subPackages = [ "./cmd/incipio" ];

        meta = with pkgs.lib; {
          description = "A simple TUI application launcher";
          homepage = "https://github.com/barab-i/incipio";
          license = licenses.mit;
          maintainers = with maintainers; [ "barab-i" ];
        };
      };

      apps.default = flake-utils.lib.mkApp {
        drv = self.packages.${system}.default;
      };
    });
}