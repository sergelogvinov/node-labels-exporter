# Make release

## Change release version

```shell
git commit --allow-empty -m "chore: release 2.0.0" -m "Release-As: 2.0.0"
```

## Update helm chart and documentation

```shell
git checkout release-please--branches--main
git tag v0.0.2

make helm-unit docs

git add .
git commit -s --amend
```
