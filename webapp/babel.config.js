// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

const presets = (modules) => [
    ['@babel/preset-env', {
        targets: {
            chrome: 66,
            firefox: 60,
            edge: 42,
            safari: 12,
        },
        modules,
        corejs: 3,
        debug: false,
        useBuiltIns: 'usage',
        shippedProposals: true,
    }],
    ['@babel/preset-react', {
        runtime: 'automatic',
    }],
    ['@babel/typescript', {
        allExtensions: true,
        isTSX: true,
    }],
];

const plugins = [
    '@babel/plugin-transform-class-properties',
    '@babel/plugin-syntax-dynamic-import',
    '@babel/plugin-transform-object-rest-spread',
];

const config = {
    presets: presets(false),
    plugins,
    env: {
        test: {
            presets: presets('auto'),
            plugins,
        },
    },
};

module.exports = config;
