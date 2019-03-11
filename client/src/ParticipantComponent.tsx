import * as React from "react";
import { Checkbox } from "./Checkbox";
import { ClassPicker } from "./ClassPicker";
import * as Conf from "./conference";
import { FormField } from "./FormField";
import { Participant } from "./types";

const staffHelp = "Select your role if you are helping to staff the " +
  "conference. If you have multiple roles and one of them is teaching " +
  "a class, then select \"Instructor\"";

const qrCodeHelp = "You can share your name, email address and phone " +
  "number with other attendees by inviting them to scan your badge " +
  "with a QR reader application on their mobile device. Select no if " +
  "you do not want a QR code printed on your name badge.";

const nicknameHelp = "If set, the nickname will be printed on your badge instead of your first name.";

const bsaNumberHelp = "We use this number when updating your official BSA training record.";

const oaBanquetHelp = "The T'Kope Kwiskwis Lodge Banquet is in \"The Grove\" of the " +
  "new Health Sciences Building at 5:30pm-9:30pm. All are welcome, including " +
  "non-OA members. The cost is $30 by October 15 and $40 after.";

type ParticipantKey = keyof Participant; // TODO: remove uses of this type.

const pleaseSelectOption = <option key="pleaseselect" value="" disabled>Please select...</option>;

function valueOption(value: string) {
  return <option key={value} value={value}>{value}</option>;
}

function valueTextOption(value: string, text: string) {
  return <option key={value} value={value}>{text}</option>;
}

const mealRequirementOptions = [
  valueTextOption("", "None"),
  valueOption("Gluten Free"),
  valueOption("Vegitarian"),
  valueOption("Vegan"),
  valueOption("Gluten Free & Vegitarian"),
  valueOption("Gluten Free & Vegan"),
];

const unitTypeOptions = [
  pleaseSelectOption,
  ...Conf.unitTypes.map((u): React.ReactNode => (<option key={u} value={u}>{u}</option>)),
];

const councilOptions = [
  pleaseSelectOption,
  ...Conf.councils.map((c): React.ReactNode => (<option key={c} value={c}>{c}</option>)),
];

const districtOptions = [
  pleaseSelectOption,
  ...Conf.districts.map((d): React.ReactNode =>
    (<option key={d.name} value={d.name}>{d.name}{d.help && ` - ${d.help}`}</option>)),
];

const qrCodeOptions = [
  valueTextOption("false", "No"),
  valueTextOption("true", "Yes"),
];

const youthOptions = [
  valueTextOption("false", "No"),
  valueTextOption("true", "Yes"),
];

const staffOptions = Conf.staffRoles.map((s): React.ReactNode => (
  <option key={s.name} value={s.name}>{s.description ? `Yes - ${s.description}` : "No"}</option>));

const oaBanquetOptions = [
  valueTextOption("false", "No"),
  valueTextOption("true", "Yes"),
];

const stateOptions = [
  pleaseSelectOption,
  valueTextOption("AL", "Alabama"),
  valueTextOption("AK", "Alaska"),
  valueTextOption("AZ", "Arizona"),
  valueTextOption("AR", "Arkansas"),
  valueTextOption("CA", "California"),
  valueTextOption("CO", "Colorado"),
  valueTextOption("CT", "Connecticut"),
  valueTextOption("DE", "Delaware"),
  valueTextOption("DC", "District of Columbia"),
  valueTextOption("FL", "Florida"),
  valueTextOption("GA", "Georgia"),
  valueTextOption("HI", "Hawaii"),
  valueTextOption("ID", "Idaho"),
  valueTextOption("IL", "Illinois"),
  valueTextOption("IN", "Indiana"),
  valueTextOption("IA", "Iowa"),
  valueTextOption("KS", "Kansas"),
  valueTextOption("KY", "Kentucky"),
  valueTextOption("LA", "Louisiana"),
  valueTextOption("ME", "Maine"),
  valueTextOption("MD", "Maryland"),
  valueTextOption("MA", "Massachusetts"),
  valueTextOption("MI", "Michigan"),
  valueTextOption("MN", "Minnesota"),
  valueTextOption("MS", "Mississippi"),
  valueTextOption("MO", "Missouri"),
  valueTextOption("MT", "Montana"),
  valueTextOption("NE", "Nebraska"),
  valueTextOption("NV", "Nevada"),
  valueTextOption("NH", "New Hampshire"),
  valueTextOption("NJ", "New Jersey"),
  valueTextOption("NM", "New Mexico"),
  valueTextOption("NY", "New York"),
  valueTextOption("NC", "North Carolina"),
  valueTextOption("ND", "North Dakota"),
  valueTextOption("OH", "Ohio"),
  valueTextOption("OK", "Oklahoma"),
  valueTextOption("OR", "Oregon"),
  valueTextOption("PA", "Pennsylvania"),
  valueTextOption("RI", "Rhode Island"),
  valueTextOption("SC", "South Carolina"),
  valueTextOption("SD", "South Dakota"),
  valueTextOption("TN", "Tennessee"),
  valueTextOption("TX", "Texas"),
  valueTextOption("UT", "Utah"),
  valueTextOption("VT", "Vermont"),
  valueTextOption("VA", "Virginia"),
  valueTextOption("WA", "Washington"),
  valueTextOption("WV", "West Virginia"),
  valueTextOption("WI", "Wisconsin"),
  valueTextOption("WY", "Wyoming"),
  <option key="1" value="" disabled>&ndash;</option>,
  valueTextOption("AS", "American Samoa"),
  valueTextOption("FM", "Federated States of Micronesia"),
  valueTextOption("GU", "Guam"),
  valueTextOption("MH", "Marshall Islands"),
  valueTextOption("MP", "Northern Mariana Islands"),
  valueTextOption("PR", "Puerto Rico"),
  valueTextOption("PW", "Palau"),
  valueTextOption("VI", "Virgin Islands"),
  <option key="2" value="" disabled>&ndash;</option>,
  valueTextOption("AA", "Armed Forces Americas"),
  valueTextOption("AE", "Armed Forces Europe"),
  valueTextOption("AP", "Armed Forces Pacific"),
  <option key="3" value="" disabled>&ndash;</option>,
  valueTextOption("AB", "Alberta"),
  valueTextOption("BC", "British Columbia"),
  valueTextOption("MB", "Manitoba"),
  valueTextOption("NB", "New Brunswick"),
  valueTextOption("NL", "Newfoundland and Labrador"),
  valueTextOption("NS", "Nova Scotia"),
  valueTextOption("NT", "Northwest Territories"),
  valueTextOption("NU", "Nunavut"),
  valueTextOption("ON", "Ontario"),
  valueTextOption("PE", "Prince Edward Island"),
  valueTextOption("QC", "Quebec"),
  valueTextOption("SK", "Saskatchewan"),
  valueTextOption("YT", "Yukon"),
];

interface ParticipantFieldProps {
  name: keyof Participant;
  label: string;
  inputType?: string;
  autoComplete?: string;
  groupClass?: string;
  placeholder?: string;
  help?: string;
  disabled?: boolean;
  invalidFeedback?: string;
  selectOptions?: React.ReactNode;
}

interface ParticipantProps {
  participant: Participant;
  validate?: boolean;
  onChange: (participant: Participant) => void;
}

export class ParticipantComponent extends React.PureComponent<ParticipantProps> {

  public render = () => {
    const props = this.props;
    const participant = props.participant;
    const invalidProps = props.validate ? Conf.validateParticipant(participant) : {};

    const formField = (options: ParticipantFieldProps): React.ReactNode => {
      const meta = Conf.participantMetadata[options.name];
      const value = meta.type.toControl(props.participant[options.name]);
      return <FormField
        value={value}
        onChange={this.handleChange}
        invalid={invalidProps[options.name]}
        convert={meta.type.fromControl}
        optional={meta.optional}
        {...options}
      />;
    };

    const marketingCheckbox = (value: string, label: string) => {
      return <Checkbox name="marketing" value={value} label={label}
        values={participant.marketing} onChange={this.handleChange} />;
    };

    // Inputs must be wrapped in <form> for browser autocomplete.

    return <form className="mt-4">
      {formField({
        name: "youth",
        label: "Are you age 20 or younger?",
        selectOptions: youthOptions,
      })}
      {formField({
        name: "staff",
        label: "Are you a member of PTC Staff?",
        help: staffHelp,
        selectOptions: staffOptions,
      })}
      <div className="form-row">
        {formField({
          name: "firstName",
          label: "First Name",
          autoComplete: "given-name",
          groupClass: "col-md-5",
          invalidFeedback: "Please provide a name.",
        })}
        {formField({
          name: "lastName",
          label: "Last Name",
          autoComplete: "family-name",
          groupClass: "col-md-5",
          invalidFeedback: "Please provide a name.",
        })}
        {formField({
          name: "suffix",
          label: "Suffix",
          autoComplete: "honorific-suffix",
          placeholder: "Jr., Sr. ...",
          groupClass: "col-md-2",
        })}
      </div>
      <div className="form-row mb-4">
        {formField({
          name: "email",
          label: "Email",
          inputType: "email",
          autoComplete: "email",
          groupClass: "col-md-6",
          invalidFeedback: "Please provide an email address.",
        })}
        {formField({
          name: "phone",
          label: "Phone",
          inputType: "tel",
          autoComplete: "tel",
          groupClass: "col-md-6",
          invalidFeedback: "Please provide a phone number.",
        })}
      </div>
      {formField({
        name: "address",
        label: "Address",
        placeholder: "123 Main St",
        autoComplete: "address-line1",
        invalidFeedback: "Please provide a street address.",
      })}
      {formField({
        name: "address2",
        label: "Address 2",
        placeholder: "Apartment, studio or floor",
        autoComplete: "address-line2",
      })}
      <div className="form-row mb-4">
        {formField({
          name: "city",
          label: "City",
          autoComplete: "address-level2",
          groupClass: "col-md-6",
          invalidFeedback: "Please provide a city.",
        })}
        {formField({
          name: "state",
          label: "State",
          autoComplete: "address-level2",
          groupClass: "col-md-4",
          selectOptions: stateOptions,
          invalidFeedback: "Please provide a state.",
        })}
        {formField({
          name: "zip",
          label: "ZIP Code",
          autoComplete: "postal-code",
          groupClass: "col-md-2",
          invalidFeedback: "Please provide a valid ZIP code.",
        })}
      </div>
      <div className="form-row">
        {formField({
          name: "council",
          label: "Council",
          groupClass: "col-md-6",
          selectOptions: councilOptions,
          invalidFeedback: "Please select your council or \"Other\".",
        })}
        {formField({
          name: "district",
          label: "District",
          disabled: !Conf.councilHasDistrict(participant.council),
          groupClass: "col-md-6",
          selectOptions: districtOptions,
          invalidFeedback: "Please select your district or \"Other\".",
        })}
      </div>
      <div className="form-row">
        {formField({
          name: "unitType",
          label: "Unit Type",
          groupClass: "col-md-6",
          selectOptions: unitTypeOptions,
          invalidFeedback: "Please select your unit type.",
        })}
        {formField({
          name: "unitNumber",
          label: "Unit Number",
          inputType: "number",
          placeholder: "Enter 0 if not associated with a unit",
          disabled: !Conf.unitTypeHasNumber(participant.unitType),
          groupClass: "col-md-6",
          invalidFeedback: "Please provide your unit number (Troop number, Pack number, ....).",
        })}
      </div>
      {formField({
        name: "bsaNumber",
        label: "BSA Member Number",
        help: bsaNumberHelp,
      })}
      {formField({
        name: "mealRequirements",
        label: "Meal Requirements",
        groupClass: "mb-4",
        selectOptions: mealRequirementOptions,
      })}
      {formField({
        name: "nickname",
        label: "Nickname for PTC Badge",
        help: nicknameHelp,
      })}
      {formField({
        name: "showQRCode",
        label: "Print QR code on PTC badge?",
        help: qrCodeHelp,
        selectOptions: qrCodeOptions,
      })}
      {formField({
        name: "oaBanquet",
        label: "Attend the Order of the Arrow Banquet",
        help: oaBanquetHelp,
        selectOptions: oaBanquetOptions,
      })}
      {formField({
        name: "scoutingYears",
        label: "How many years have you been in scouting?",
        inputType: "number",
        groupClass: "mb-4",
      })}
      <div className="form-group mb-4">
        <div>How did you hear about the PTC?</div>
        <div className="small mb-2 text-muted">Select all that apply.</div>
        <div className="form-row">
          <div className="col">
            {marketingCheckbox("website", "Council website")}
            {marketingCheckbox("district", "District or Roundtable")}
            {marketingCheckbox("unit", "Unit")}
          </div>
          <div className="col">
            {marketingCheckbox("woodBadge", "Wood Bage")}
            {marketingCheckbox("eTotem", "eTotem")}
            {marketingCheckbox("attended", "Attended before")}
          </div>
        </div>
        <input type="text"
          name="marketingOther"
          value={participant.marketingOther}
          className="form-control"
          placeholder="Other"
          onChange={this.handleMarketingOtherChange} />
      </div>
      <ClassPicker
        classes={participant.classes}
        instructorClasses={participant.instructorClasses}
        instructor={participant.staff === "instructor"}
        onChange={this.handleClasssesChange}
        className="mb-4" />
    </form>;
  }

  private handleChange = (name: keyof Participant, value: any) => {
    const p = { ...this.props.participant };
    p[name] = value;
    this.props.onChange(p);
  }

  private handleMarketingOtherChange = (ev: React.ChangeEvent<HTMLInputElement>) => {
    this.handleChange("marketingOther", ev.currentTarget.value);
  }

  private handleClasssesChange = (classes: number[], instructorClasses: number[]) => {
    const p = { ...this.props.participant };
    p.classes = classes;
    p.instructorClasses = instructorClasses;
    this.props.onChange(p);
  }

}
