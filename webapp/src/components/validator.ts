// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export default class Validator {
    components: Map<string, () => boolean> = new Map();

    addComponent = (key: string, validateField: () => boolean) => {
        this.components.set(key, validateField);
    };

    removeComponent = (key: string) => {
        this.components.delete(key);
    };

    validate = () => {
        return Array.from(this.components.values()).reduce((accum, validateField) => {
            return validateField() && accum;
        }, true);
    };
}
