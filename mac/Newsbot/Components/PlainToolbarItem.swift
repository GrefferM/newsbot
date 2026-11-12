import SwiftUI

struct PlainToolbarItem<Label: View>: ToolbarContent {
    let placement: ToolbarItemPlacement
    @ViewBuilder let label: () -> Label

    var body: some ToolbarContent {
        if #available(macOS 26.0, *) {
            ToolbarItem(placement: placement, content: label)
                .sharedBackgroundVisibility(.hidden)
        } else {
            ToolbarItem(placement: placement, content: label)
        }
    }
}

struct ToolbarGap: ToolbarContent {
    var body: some ToolbarContent {
        if #available(macOS 26.0, *) {
            ToolbarSpacer(.flexible)
        } else {
            ToolbarItem {
                Spacer()
            }
        }
    }
}
