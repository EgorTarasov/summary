// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export interface PluginRegistry {
    registerPostTypeComponent(typeName: string, component: React.ElementType);
    registerRootComponent(component: React.ComponentType);
    registerRightHandSidebarComponent(component: React.ComponentType, title: string);
    registerPostDropdownMenuComponent(component: React.ComponentType);
    registerChannelHeaderButtonAction(component: React.ComponentType, title: string, tooltip: string);
    registerPostMenuAction(text: string, action: (postId: string, channelId: string) => void, filter?: (post: any) => boolean);

    // Add more if needed from https://developers.mattermost.com/extend/plugins/webapp/reference
}
