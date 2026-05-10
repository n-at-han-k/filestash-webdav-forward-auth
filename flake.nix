{
  description = "Filestash fork with forward-auth WebDAV";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gnumake
            pkg-config
            curl

            brotli
            libjpeg
            libtiff
            libpng
            libwebp
            libraw
            libheif
            giflib
            vips
            sqlite
            ffmpeg
          ];
        };
      });
}
