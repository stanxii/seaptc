import * as React from "react";
import { uniqueID } from "./conference";

export interface CheckboxProps {
  name: string;
  label: string;
  value: string; // Value for this checkbox
  values?: string[]; // Currently checked values
  onChange(name: string, values: string[]): void;
}

export class Checkbox extends React.PureComponent<CheckboxProps> {

  public render = () => {
    const { values = [], value, label, name } = this.props;
    const id = uniqueID(name + value);
    const checked = values.indexOf(value) >= 0;
    return <div className="form-check">
      <input id={id} className="form-check-input" type="checkbox" onChange={this.handleChange} checked={checked} />
      <label className="form-check-label" htmlFor={id}>{label}</label>
    </div>;
  }

  private handleChange = (ev: React.ChangeEvent<HTMLInputElement>) => {
    const { values = [], value, name, onChange } = this.props;
    const newValues = [...values];
    const i = newValues.indexOf(value);
    if (ev.currentTarget.checked) {
      if (i < 0) {
        newValues.push(value);
        newValues.sort();
      }
    } else {
      if (i >= 0) {
        newValues.splice(i, 1);
      }
    }
    onChange(name, newValues);
  }

}
