// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable max-nested-callbacks, @typescript-eslint/no-non-null-assertion, max-lines, @typescript-eslint/no-explicit-any */

import {describe, test, expect, jest, beforeEach, afterEach} from '@jest/globals';

import ActionTypes, {RHSViewType, RHSViewTypeValue} from './action_types';

// Types for test mocks
interface MockStore {
    dispatch: jest.Mock;
    getState: jest.Mock;
}

interface PopoutListeners {
    onMessageFromPopout: (callback: (channel: string) => void) => void;
    sendToPopout: (channel: string, data: unknown) => void;
}

type PopoutListenerCallback = (teamName: string, channelName: string, listeners: PopoutListeners) => void;

interface PluginInterface {
    initialize: (registry: MockRegistry, store: MockStore) => Promise<void>;
    deinitialize: () => void;
}

interface MockRegistry {
    registerReducer: jest.Mock;
    registerLeftSidebarHeaderComponent: jest.Mock;
    registerBottomTeamSidebarComponent: jest.Mock;
    registerPopoverUserAttributesComponent: jest.Mock;
    registerRootComponent: jest.Mock;
    registerPostDropdownMenuAction: jest.Mock;
    registerLinkTooltipComponent: jest.Mock;
    registerSlashCommandWillBePostedHook: jest.Mock;
    registerRightHandSidebarComponent: jest.Mock;
    registerWebSocketEventHandler: jest.Mock;
    registerReconnectHandler: jest.Mock;
    registerAppBarComponent: jest.Mock;
    registerRHSPluginPopoutListener: jest.Mock;
}

interface TestWindow {
    registerPlugin: jest.Mock | ((id: string, plugin: PluginInterface) => void);
    WebappUtils?: {
        popouts?: {
            isPopoutWindow: jest.Mock | (() => boolean);
            onMessageFromParent: jest.Mock | ((callback: (channel: string, state: unknown) => void) => void);
            sendToParent: jest.Mock | ((channel: string, data?: unknown) => void);
        };
    };
}

// Typed reference to window for testing
const testWindow = window as unknown as TestWindow;

// Mock the manifest
jest.mock('./manifest', () => ({
    __esModule: true,
    default: {
        id: 'com.github.manland.mattermost-plugin-gitlab',
        version: '1.0.0',
    },
}));

// Mock the Client
jest.mock('./client', () => {
    const mockLHSData = {reviews: [], yourAssignedPrs: [], yourAssignedIssues: [], todos: []};
    const mockSubscriptions = {subscriptions: []};
    return {
        __esModule: true,
        default: {
            setServerRoute: jest.fn(),
            getLHSData: jest.fn(() => Promise.resolve(mockLHSData)),
            getChannelSubscriptions: jest.fn(() => Promise.resolve(mockSubscriptions)),
        },
    };
});

// Mock selectors
jest.mock('./selectors', () => ({
    getPluginServerRoute: jest.fn(() => '/plugins/com.github.manland.mattermost-plugin-gitlab'),
    getConnected: jest.fn(() => true),
    getPluginState: jest.fn(() => ({
        rhsViewType: 'subscriptions',
        rhsState: 'reviews',
        connected: true,
    })),
}));

// Mock mattermost-redux selectors
jest.mock('mattermost-redux/selectors/entities/general', () => ({
    getConfig: jest.fn(() => ({SiteURL: 'http://localhost:8065'})),
}));

jest.mock('mattermost-redux/selectors/entities/common', () => ({
    getCurrentChannelId: jest.fn(() => 'channel-id-123'),
}));

jest.mock('mattermost-redux/selectors/entities/posts', () => ({
    getPost: jest.fn(() => ({id: 'post-id'})),
}));

jest.mock('mattermost-redux/utils/post_utils', () => ({
    isSystemMessage: jest.fn(() => false),
}));

// Mock components
jest.mock('./components/sidebar_header', () => ({__esModule: true, default: () => null}));
jest.mock('./components/team_sidebar', () => ({__esModule: true, default: () => null}));
jest.mock('./components/user_attribute', () => ({__esModule: true, default: () => null}));
jest.mock('./components/post_options/create_issue', () => ({__esModule: true, default: 'Create Issue'}));
jest.mock('./components/post_options/attach_comment_to_issue', () => ({__esModule: true, default: 'Attach Comment'}));
jest.mock('./components/modals/create_issue/create_issue_modal', () => ({__esModule: true, default: () => null}));
jest.mock('./components/modals/attach_comment_to_issue/attach_comment_to_issue_modal', () => ({
    __esModule: true,
    default: () => null,
}));
jest.mock('./components/gitlab_rhs', () => ({__esModule: true, default: () => null}));
jest.mock('./components/link_tooltip', () => ({__esModule: true, default: () => null}));

// Mock reducers
jest.mock('./reducers', () => ({__esModule: true, default: () => ({})}));

// Mock hooks
jest.mock('./hooks', () => ({
    __esModule: true,
    default: jest.fn().mockImplementation(() => ({
        slashCommandWillBePostedHook: jest.fn(),
    })),
}));

// Mock websocket handlers
jest.mock('./websocket', () => ({
    handleConnect: jest.fn(() => jest.fn()),
    handleDisconnect: jest.fn(() => jest.fn()),
    handleReconnect: jest.fn(() => jest.fn()),
    handleRefresh: jest.fn(() => jest.fn()),
    handleOpenCreateIssueModal: jest.fn(() => jest.fn()),
    handleChannelSubscriptionsUpdated: jest.fn(() => jest.fn()),
}));

// Mock actions
const mockSetShowRHSAction = jest.fn(() => ({type: 'SET_SHOW_RHS_ACTION'}));
const mockSetRHSViewType = jest.fn((viewType: RHSViewTypeValue) => ({
    type: ActionTypes.SET_RHS_VIEW_TYPE,
    viewType,
}));
const mockUpdateRHSState = jest.fn((state: string) => ({type: ActionTypes.UPDATE_RHS_STATE, state}));
const mockGetLHSData = jest.fn(() => () => Promise.resolve({data: {}}));
const mockGetChannelSubscriptions = jest.fn((channelId: string) => () => Promise.resolve({data: [], channelId}));
const mockGetConnected = jest.fn(() => () => Promise.resolve({data: {}}));

jest.mock('./actions', () => ({
    getConnected: () => mockGetConnected(),
    openAttachCommentToIssueModal: jest.fn(() => ({type: 'OPEN_ATTACH_COMMENT_MODAL'})),
    openCreateIssueModal: jest.fn(() => ({type: 'OPEN_CREATE_ISSUE_MODAL'})),
    setShowRHSAction: () => mockSetShowRHSAction(),
    getLHSData: () => mockGetLHSData(),
    updateRHSState: (state: string) => mockUpdateRHSState(state),
    setRHSViewType: (viewType: RHSViewTypeValue) => mockSetRHSViewType(viewType),
    getChannelSubscriptions: (channelId: string) => mockGetChannelSubscriptions(channelId),
}));

// Helper to create mock store
function createMockStore(): MockStore {
    return {
        dispatch: jest.fn((action) => {
            if (typeof action === 'function') {
                return action();
            }
            return action;
        }),
        getState: jest.fn(() => ({
            'plugins-com.github.manland.mattermost-plugin-gitlab': {
                rhsViewType: RHSViewType.SUBSCRIPTIONS,
                rhsState: 'reviews',
                connected: true,
            },
        })),
    };
}

// Helper to create mock registry
function createMockRegistry(
    showRHSPlugin: jest.Mock,
    toggleRHSPlugin: jest.Mock,
    capturePopoutListener: (callback: PopoutListenerCallback) => void,
): MockRegistry {
    return {
        registerReducer: jest.fn(),
        registerLeftSidebarHeaderComponent: jest.fn(),
        registerBottomTeamSidebarComponent: jest.fn(),
        registerPopoverUserAttributesComponent: jest.fn(),
        registerRootComponent: jest.fn(),
        registerPostDropdownMenuAction: jest.fn(),
        registerLinkTooltipComponent: jest.fn(),
        registerSlashCommandWillBePostedHook: jest.fn(),
        registerRightHandSidebarComponent: jest.fn(() => ({
            showRHSPlugin,
            toggleRHSPlugin,
        })),
        registerWebSocketEventHandler: jest.fn(),
        registerReconnectHandler: jest.fn(),
        registerAppBarComponent: jest.fn(),
        registerRHSPluginPopoutListener: jest.fn((pluginId: string, callback: PopoutListenerCallback) => {
            capturePopoutListener(callback);
        }) as any,
    };
}

// Helper to get PluginClass
function getPluginClass(): new () => PluginInterface {
    let PluginClass: new () => PluginInterface;

    jest.isolateModules(() => {
        testWindow.registerPlugin = jest.fn((id: string, plugin: PluginInterface) => {
            PluginClass = plugin.constructor as new () => PluginInterface;
        });
        require('./index'); // eslint-disable-line global-require
    });

    return PluginClass!;
}

describe('GitLab Plugin Initialization', () => {
    let mockStore: MockStore;
    let mockRegistry: MockRegistry;
    let mockShowRHSPlugin: jest.Mock;
    let mockToggleRHSPlugin: jest.Mock;
    let popoutListenerCallback: PopoutListenerCallback | null = null;

    beforeEach(() => {
        jest.clearAllMocks();

        mockShowRHSPlugin = jest.fn();
        mockToggleRHSPlugin = jest.fn(() => ({type: 'TOGGLE_RHS'}));

        mockStore = createMockStore();
        mockRegistry = createMockRegistry(
            mockShowRHSPlugin,
            mockToggleRHSPlugin,
            (callback) => {
                popoutListenerCallback = callback;
            },
        );

        // Reset window mock
        delete testWindow.WebappUtils;
        testWindow.registerPlugin = jest.fn();
    });

    afterEach(() => {
        popoutListenerCallback = null;
    });

    describe('Plugin Registration', () => {
        test('should register plugin with registerPlugin', async () => {
            jest.isolateModules(() => {
                require('./index'); // eslint-disable-line global-require
            });

            expect(testWindow.registerPlugin).toHaveBeenCalledWith(
                'com.github.manland.mattermost-plugin-gitlab',
                expect.any(Object),
            );
        });
    });

    describe('initialize()', () => {
        test('should register reducer and RHS component', async () => {
            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            expect(mockRegistry.registerReducer).toHaveBeenCalled();
            expect(mockRegistry.registerRightHandSidebarComponent).toHaveBeenCalledWith(expect.anything(), 'GitLab');
        });

        test('should set showRHSAction and call getConnected', async () => {
            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            expect(mockSetShowRHSAction).toHaveBeenCalled();
            expect(mockGetConnected).toHaveBeenCalled();
        });

        test('should register WebSocket event handlers', async () => {
            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            expect(mockRegistry.registerWebSocketEventHandler).toHaveBeenCalledWith(expect.stringContaining('gitlab_connect'), expect.any(Function));
            expect(mockRegistry.registerWebSocketEventHandler).toHaveBeenCalledWith(expect.stringContaining('gitlab_disconnect'), expect.any(Function));
            expect(mockRegistry.registerWebSocketEventHandler).toHaveBeenCalledWith(expect.stringContaining('gitlab_refresh'), expect.any(Function));
            expect(mockRegistry.registerWebSocketEventHandler).toHaveBeenCalledWith(expect.stringContaining('create_issue'), expect.any(Function));
            expect(mockRegistry.registerWebSocketEventHandler).toHaveBeenCalledWith(expect.stringContaining('gitlab_channel_subscriptions_updated'), expect.any(Function));
        });

        test('should register App Bar component when available', async () => {
            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            expect(mockRegistry.registerAppBarComponent).toHaveBeenCalledWith(
                expect.stringContaining('app-bar-icon.png'),
                expect.any(Function),
                'GitLab',
            );
        });
    });

    describe('showSubscriptionsRHS', () => {
        test('should set RHS view type to SUBSCRIPTIONS and toggle RHS', async () => {
            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            const showSubscriptionsRHS = mockRegistry.registerAppBarComponent.mock.calls[0][1] as () => void;
            showSubscriptionsRHS();

            expect(mockSetRHSViewType).toHaveBeenCalledWith(RHSViewType.SUBSCRIPTIONS);
            expect(mockStore.dispatch).toHaveBeenCalledWith(mockToggleRHSPlugin());
        });
    });

    describe('RHS Plugin Popout Listener', () => {
        test('should register popout listener when available', async () => {
            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            expect(mockRegistry.registerRHSPluginPopoutListener).toHaveBeenCalledWith(
                'com.github.manland.mattermost-plugin-gitlab',
                expect.any(Function),
            );
        });

        test('should respond to GET_POPOUT_STATE with current state', async () => {
            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            expect(popoutListenerCallback).not.toBeNull();

            const mockSendToPopout = jest.fn();
            let onMessageCallback: ((channel: string) => void) | null = null;

            const mockListeners: PopoutListeners = {
                onMessageFromPopout: jest.fn((callback: (channel: string) => void) => {
                    onMessageCallback = callback;
                }),
                sendToPopout: mockSendToPopout,
            };

            popoutListenerCallback!('team-name', 'channel-name', mockListeners);
            onMessageCallback!('GET_POPOUT_STATE');

            expect(mockSendToPopout).toHaveBeenCalledWith('SEND_POPOUT_STATE', {
                rhsViewType: RHSViewType.SUBSCRIPTIONS,
                rhsState: 'reviews',
                channelId: 'channel-id-123',
            });
        });

        test('should not respond to other message channels', async () => {
            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            const mockSendToPopout = jest.fn();
            let onMessageCallback: ((channel: string) => void) | null = null;

            popoutListenerCallback!('team-name', 'channel-name', {
                onMessageFromPopout: jest.fn((callback: (channel: string) => void) => {
                    onMessageCallback = callback;
                }),
                sendToPopout: mockSendToPopout,
            });

            onMessageCallback!('SOME_OTHER_MESSAGE');
            expect(mockSendToPopout).not.toHaveBeenCalled();
        });
    });

    describe('Popout Window Behavior', () => {
        interface PopoutState {
            rhsViewType: RHSViewTypeValue;
            rhsState: string;
            channelId: string;
        }

        let onMessageFromParentCallback: ((channel: string, state: PopoutState) => void) | null = null;

        beforeEach(() => {
            testWindow.WebappUtils = {
                popouts: {
                    isPopoutWindow: jest.fn(() => true),
                    onMessageFromParent: jest.fn((callback: (channel: string, state: PopoutState) => void) => {
                        onMessageFromParentCallback = callback;
                    }),
                    sendToParent: jest.fn(),
                },
            };
        });

        afterEach(() => {
            onMessageFromParentCallback = null;
        });

        test('should fetch LHS data and request state from parent', async () => {
            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            expect(mockGetLHSData).toHaveBeenCalled();
            expect(testWindow.WebappUtils?.popouts?.sendToParent).toHaveBeenCalledWith('GET_POPOUT_STATE');
        });

        test('should handle SEND_POPOUT_STATE message and update state', async () => {
            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            expect(onMessageFromParentCallback).not.toBeNull();

            const popoutState: PopoutState = {
                rhsViewType: RHSViewType.SIDEBAR_RIGHT,
                rhsState: 'todos',
                channelId: 'new-channel-id',
            };

            onMessageFromParentCallback!('SEND_POPOUT_STATE', popoutState);

            expect(mockSetRHSViewType).toHaveBeenCalledWith(RHSViewType.SIDEBAR_RIGHT);
            expect(mockUpdateRHSState).toHaveBeenCalledWith('todos');
            expect(mockStore.dispatch).toHaveBeenCalledWith({
                type: ActionTypes.SET_POPOUT_CHANNEL_ID,
                channelId: 'new-channel-id',
            });
            expect(mockGetChannelSubscriptions).toHaveBeenCalledWith('new-channel-id');
        });

        test('should handle partial popout state (without rhsState or channelId)', async () => {
            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            // Test without rhsState
            mockUpdateRHSState.mockClear();
            onMessageFromParentCallback!('SEND_POPOUT_STATE', {
                rhsViewType: RHSViewType.SUBSCRIPTIONS,
                rhsState: '',
                channelId: 'channel-123',
            });
            expect(mockUpdateRHSState).not.toHaveBeenCalled();

            // Test without channelId
            mockStore.dispatch.mockClear();
            mockGetChannelSubscriptions.mockClear();
            onMessageFromParentCallback!('SEND_POPOUT_STATE', {
                rhsViewType: RHSViewType.SIDEBAR_RIGHT,
                rhsState: 'reviews',
                channelId: '',
            });
            expect(mockStore.dispatch).not.toHaveBeenCalledWith(
                expect.objectContaining({type: ActionTypes.SET_POPOUT_CHANNEL_ID}),
            );
        });

        test('should ignore non-SEND_POPOUT_STATE messages', async () => {
            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            mockSetRHSViewType.mockClear();
            onMessageFromParentCallback!('SOME_OTHER_MESSAGE', {
                rhsViewType: RHSViewType.SIDEBAR_RIGHT,
                rhsState: '',
                channelId: '',
            });

            expect(mockSetRHSViewType).not.toHaveBeenCalled();
        });
    });

    describe('Non-Popout Window Behavior', () => {
        beforeEach(() => {
            testWindow.WebappUtils = {
                popouts: {
                    isPopoutWindow: jest.fn(() => false),
                    onMessageFromParent: jest.fn(),
                    sendToParent: jest.fn(),
                },
            };
        });

        test('should NOT send messages to parent when not in popout', async () => {
            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            expect(testWindow.WebappUtils?.popouts?.sendToParent).not.toHaveBeenCalled();
        });
    });

    describe('deinitialize()', () => {
        test('should remove click event listener', async () => {
            const removeEventListenerSpy = jest.spyOn(document, 'removeEventListener');

            const PluginClass = getPluginClass();
            const plugin = new PluginClass();
            await plugin.initialize(mockRegistry, mockStore);

            plugin.deinitialize();

            expect(removeEventListenerSpy).toHaveBeenCalledWith('click', expect.any(Function));
            removeEventListenerSpy.mockRestore();
        });
    });
});
