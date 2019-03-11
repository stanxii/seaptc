import * as React from "react";
import * as Conf from "./conference";
import { ParticipantComponent } from "./ParticipantComponent";
import { Participant, Registration } from "./types";

interface RegistrationProps {
  registration: Registration;
}

interface RegistrationState {
  registration: Registration;
  currentParticipant: number;
  validate: boolean;
}

export class RegistrationComponent extends React.Component<RegistrationProps, RegistrationState> {
  constructor(props: RegistrationProps) {
    super(props);
    this.state = { registration: props.registration, currentParticipant: 0, validate: false };
  }

  public render() {
    const state = this.state;
    return <div>
      <div className="form-check">
        <input className="form-check-input" type="checkbox"
          onChange={this.handleValidateChange} checked={state.validate || false} />
        <label className="form-check-label">Validate</label>
      </div>
      <ParticipantComponent
        validate={state.validate}
        participant={state.registration.participants[state.currentParticipant]}
        onChange={this.handleParticipantChange} />
    </div>;
  }

  private handleValidateChange = (ev: React.ChangeEvent<HTMLInputElement>): void => {
    const validate = ev.currentTarget.checked;
    this.setState((prevState, props) => ({ validate }));
  }

  private logDiff = (prev: Participant, next: Participant) => {
    const diffs: { [k: string]: any } = {};
    Object.keys(Conf.participantMetadata).forEach((name: keyof Participant) => {
      if (prev[name] !== next[name]) {
        diffs[name] = { Prev: prev[name], Next: next[name] };
      }
    });
    console.table(diffs);
  }

  private handleParticipantChange = (p: Participant) => {
    p = Conf.fixupParticipant(p);
    this.setState((prevState, props) => {
      const participants = [...prevState.registration.participants];
      for (let i = 0; i < participants.length; i++) {
        if (participants[i].id === p.id) {
          this.logDiff(participants[i], p);
          participants[i] = p;
        }
      }
      const registration = { ...prevState.registration };
      registration.participants = participants;
      return { registration };
    });
  }

}
