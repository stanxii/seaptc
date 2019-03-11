// Object property values are not null or undefined. A distinguished value is used
// in cases where it's important to distinguish no value from an actual value. For
// example, "floatgt0" uses -1 as the not set value.

function missing(t: string, p: string): any {
    console.error("%s.%s not set", t, p);
}

interface ValueType<V> {
    name: string;
    initialValue: () => V; // initial value is function to handle array an object values.
    fromControl: (s: string) => any | undefined;
    toControl: (v: any) => string;
}

export let stringType: ValueType<string> = {
    fromControl: (s: string): string => (s),
    initialValue: (): string => (""),
    name: "string",
    toControl: (v: string) => (v),
};

export let intType: ValueType<number> = {
    fromControl: (s: string): number | undefined => {
        const result = parseInt(s, 10);
        return isNaN(result) ? undefined : result;
    },
    initialValue: (): number => (0),
    name: "int",
    toControl: (v: number) => (v.toString()),
};

export let floatType: ValueType<number> = {
    fromControl: (s: string): number | undefined => {
        const result = parseFloat(s);
        return isNaN(result) ? undefined : result;
    },
    initialValue: (): number => (0),
    name: "float",
    toControl: (v: number) => (v.toString()),
};

export let booleanType: ValueType<boolean> = {
    fromControl: (s: string): boolean | undefined => {
        return s === "true";
    },
    initialValue: (): boolean => (false),
    name: "boolean",
    toControl: (v: boolean) => (v.toString()),
};

export let floatgt0Type: ValueType<number> = {
    fromControl: (s: string): number | undefined => {
        s = s.trim();
        if (s === "") {
            return -1;
        }
        const result = parseFloat(s);
        return isNaN(result) || result < 0 ? undefined : result;
    },
    initialValue: (): number => (-1),
    name: "floatgt0",
    toControl: (v: number) => (v < 0 ? "" : v.toString()),
};

export let stringArrayType: ValueType<string[]> = {
    fromControl: (s: string): string[] | undefined => (missing("stringArray", "fromControl")),
    initialValue: (): string[] => ([]),
    name: "stringArray",
    toControl: (v: string[]): string => (missing("stringArray", "toControl")),
};

export let intArrayType: ValueType<number[]> = {
    fromControl: (s: string): number[] | undefined => (missing("intArray", "fromControl")),
    initialValue: (): number[] => ([]),
    name: "intArray",
    toControl: (v: number[]): string => (missing("intArray", "toControl")),
};

export let otherType: ValueType<any> = {
    fromControl: (s: string): any | undefined => (missing("other", "fromControl")),
    initialValue: (): any => (missing("other", "initialValue")),
    name: "other",
    toControl: (v: any): string => (missing("other", "toControl")),
};

interface PropertyMetadata<T> {
    initialValue?: () => T;  // initial value is function to handle array an object values.
    optional?: boolean; // display field as optional in UI
    type: ValueType<T>;
    validation?: "truthy";
}

export type ObjectMetadata<T> = { [K in keyof T]-?: PropertyMetadata<T[K]> };

export function newObject<T>(objectMetadata: ObjectMetadata<T>): T {
    const o: { [k: string]: any } = {};
    Object.keys(objectMetadata).forEach((name) => {
        const meta = objectMetadata[name as keyof T];
        o[name] = meta.initialValue ? meta.initialValue() : meta.type.initialValue();
    });
    return o as T;
}

export type InvalidProperties<T> = { [P in keyof T]?: boolean };

export function validateObject<T>(objectMetadata: ObjectMetadata<T>, o: T): InvalidProperties<T> {
    const invalid: { [k: string]: boolean } = {};
    Object.keys(objectMetadata).forEach((name) => {
        const meta = objectMetadata[name as keyof T];
        if (meta.validation === "truthy" && !o[name as keyof T]) {
            invalid[name] = true;
        }
    });
    return invalid as InvalidProperties<T>;
}
