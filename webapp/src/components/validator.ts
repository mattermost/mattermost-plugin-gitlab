export default class Validator {
    components: Map<any, any>
    constructor() {
        // Our list of components we have to validate before allowing a submit action.
        this.components = new Map();
    }

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
