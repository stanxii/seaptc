import * as React from "react";
import { uniqueID } from "./conference";

interface FormFieldProps {
    name: string;
    value: string;
    label: string;
    convert: (s: string) => any;
    autoComplete?: string;
    placeholder?: string;
    groupClass?: string;
    help?: string;
    invalid?: boolean;
    disabled?: boolean;
    optional?: boolean;
    invalidFeedback?: string;
    inputType?: string; // <input type=?>
    selectOptions?: React.ReactNode;
    onChange: (name: string, value: any) => void;
}

export class FormField extends React.PureComponent<FormFieldProps> {

    public render = (): React.ReactNode => {
        const props = this.props;
        const id = uniqueID(props.name);
        const className = `form-control${props.invalid ? " is-invalid" : ""}`;
        let control: React.ReactNode;
        if (props.selectOptions) {
            let selectOptions = props.selectOptions;
            if (props.disabled && !props.value) {
                // Hide "Please select..."
                selectOptions = <></>;
            }
            control = <select value={props.value}
                autoComplete={props.autoComplete}
                className={className}
                disabled={props.disabled}
                onChange={this.handleChange}>
                {selectOptions}
            </select>;
        } else {
            let placeholder = props.placeholder;
            if (props.disabled) {
                placeholder = "";
            }
            control = <input
                value={props.value}
                autoComplete={props.autoComplete}
                className={className}
                disabled={props.disabled}
                onChange={this.handleChange}
                placeholder={placeholder}
                type={props.inputType || "text"}
            />;
        }
        return <div key={name} className={`form-group ${props.groupClass || ""}`}>
            <label htmlFor={id}>{props.label}
                {props.optional && <small className="text-muted"> (Optional)</small>}
            </label>
            {control}
            {props.help && <small className="form-text text-muted">{props.help}</small>}
            {props.invalidFeedback && <div className="invalid-feedback">{props.invalidFeedback}</div>}
        </div>;
    }

    private handleChange = (ev: React.ChangeEvent<HTMLInputElement> & React.ChangeEvent<HTMLSelectElement>): void => {
        const value = this.props.convert(ev.currentTarget.value);
        if (value === undefined) {
            return;
        }
        this.props.onChange(this.props.name, value);
    }
}
