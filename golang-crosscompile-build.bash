#!/bin/bash
# Orignal https://gist.github.com/jmervine/7d3f455e923cf2ac3c9e
# usage: ./golang-crosscompile-build.bash

#Clear Sceeen
printf "\033c"

# Get Version out of target then replace . with _
versiond=$(go run *.go -version)
version=${versiond//./_}
#Remove White Space
version=${version// /}
versiond=${versiond// /}
platforms="windows/386 windows/amd64 darwin/arm64"
printf " ---- Building Supportworks Request Import $versiond ---- \n"
rm -rf "release/"
rm -rf "builds/"
mkdir release
printf "\n"
for platform in ${platforms}
do
    split=(${platform//\// })
    goos=${split[0]}
    os=${split[0]}
    goarch=${split[1]}
    arch=${split[1]}
    output=goSWRequestImport
    package=goSWRequestImport
    # add exe to windows output
    [[ "windows" == "$goos" ]] && output="$output.exe"
    [[ "windows" == "$goos" ]] && os="win"
    [[ "386" == "$goarch" ]] && arch="x86"
    [[ "amd64" == "$goarch" ]] && arch="x64"

    printf "Platform: $goos - $goarch \n"

    destination="builds/$goos/$goarch/$output"

    printf "Go Build\n"
    GOOS=$goos GOARCH=$goarch go build  -o $destination $target

    printf "Copy Source Files\n"
    #Copy Source to Build Dir
    cp LICENSE.md "builds/$goos/$goarch/LICENSE.md"
    cp README.md "builds/$goos/$goarch/README.md"
    cp conf.json "builds/$goos/$goarch/conf.json"

    printf "Build Zip \n"
    cd "builds/$goos/$goarch/"
    if [ $os == "darwin" ]; then
        os="osx"
    fi
    zip -r "${package}_${os}_${arch}_v${version}.zip" $output LICENSE.md README.md conf.json > /dev/null
    cp "${package}_${os}_${arch}_v${version}.zip" "../../../release/${package}_${os}_${arch}_v${version}.zip"
    cd "../../../"
    printf "\n"
done
printf "Clean Up \n"
rm -rf "builds/"
printf "Build Complete \n"
printf "\n"