// swift-tools-version: 6.0

import PackageDescription

let package = Package(
    name: "NewsbotCore",
    platforms: [.macOS(.v14)],
    products: [
        .library(name: "NewsbotCore", targets: ["NewsbotCore"])
    ],
    targets: [
        .target(name: "NewsbotCore"),
        .testTarget(name: "NewsbotCoreTests", dependencies: ["NewsbotCore"]),
    ]
)
