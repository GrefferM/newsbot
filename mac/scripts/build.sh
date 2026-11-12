#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

xcodebuild -project Newsbot.xcodeproj -scheme Newsbot -configuration Release -derivedDataPath .build/xcode -quiet build

rm -rf .build/Newsbot.app
cp -R .build/xcode/Build/Products/Release/Newsbot.app .build/Newsbot.app
strip -S .build/Newsbot.app/Contents/MacOS/Newsbot
codesign --force --sign - --preserve-metadata=entitlements,flags,runtime .build/Newsbot.app 2>/dev/null

echo .build/Newsbot.app
