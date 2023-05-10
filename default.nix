{ pkgs ? import <nixpkgs> { } }:
pkgs.mkShell {
  # nativeBuildInputs is usually what you want -- tools you need to run
  nativeBuildInputs = with pkgs; [
    # pkgs.go
    # pkgs.fish
    go-migrate
    go-task
    air
    (callPackage ./sqlboiler.nix { })
  ];
}
