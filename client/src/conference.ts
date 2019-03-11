import * as data from "./data";
import * as Props from "./props";
import { Participant, Registration } from "./types";

export const numSessions = 6;

export interface Class {
  num: number;
  len: number;
  title: string;
  description: string;
  mask: number;
  startSession: number;
  fullTitle: string;
  freeTime: boolean;
}

export let classes: Class[] = data.classes.map((c): Class => {
  const startSession = Math.floor(c.num / 100) - 1;

  let mask = 0;
  for (let i = 0; i < c.len; i++) {
    mask |= 1 << (startSession + i);
  }

  let fullTitle: string;

  if (c.freeTime) {
    fullTitle = "No Class";
  } else {
    const sessions = (c.len <= 1)
      ? `(Session ${startSession + 1})`
      : `(Sessons ${startSession + 1} – ${startSession + c.len})`;
    fullTitle = `${c.num}: ${c.title} ${sessions}`;
  }

  return {
    freeTime: c.freeTime || false,
    fullTitle,
    len: c.len,
    mask,
    num: c.num,
    startSession,
    title: c.title,
    description: "",
  };
});

const badClass: Class = {
  freeTime: false,
  fullTitle: "",
  len: -1,
  mask: 0,
  num: 0,
  startSession: -1,
  title: "",
  description: "",
};

export function lookupClass(num: number): Class {
  // binary search
  let low = 0;
  let high = classes.length - 1;
  while (low <= high) {
    const mid = low + Math.floor((high - low) / 2);
    const n = classes[mid].num;
    if (n === num) {
      return classes[mid];
    } else if (n < num) {
      low = mid + 1;
    } else {
      high = mid - 1;
    }
  }
  return badClass;
}

export const unitTypes: string[] = [
  "Cub Pack",
  "Scout Troop",
  "Venturing Crew",
  "Sea Scout Ship",
  "District",
  "Council",
  "Other",
];

export function unitTypeHasNumber(unitType?: string): boolean {
  return unitType === ""
    || unitType === "Cub Pack"
    || unitType === "Scout Troop"
    || unitType === "Venturing Crew"
    || unitType === "Sea Scout Ship";
}

export const councils: string[] = [
  "Chief Seattle",
  "Mount Baker",
  "Pacific Harbors",
  "Grand Columbia",
  "Inland Northwest",
  "Cascade Pacific",
  "Other",
];

export function councilHasDistrict(council?: string): boolean {
  return council === "" || council === "Chief Seattle";
}

export const districts: Array<{ name: string, help?: string }> = [
  { name: "Council" },
  { name: "Other", help: "Not in Chief Seattle Council or don\"t know" },
  { name: "Alpine", help: "Cougar Mountain, Fall City, Issaquah,"
    + " North Bend, Sammamish Plateau, Snoqualmie, Renton Highlands, and Newcastle" },
  { name: "Aquila", help: "Burien, Des Moines, Normandy Park, Sea Tac, Tukwila,"
    + " Vashon Island, West Seattle, White Center" },
  { name: "Aurora", help: "Lake Forest Park, North Seattle, Shoreline" },
  { name: "Cascade", help: "Bellevue, Clyde Hill, Hunts Point, Medina, Mercer Island and Yarrow Point" },
  { name: "Foothills", help: "Auburn, Black Diamond, Covington, Maple Valley, Pacific" },
  { name: "Green River", help: "Kent, Newcastle, Renton, Skyway" },
  { name: "Mt Olympus", help: "Clallam and Jefferson counties" },
  { name: "North Lakes", help: "Bothell, Carnation, Duvall, Kenmore, Woodinville" },
  { name: "Orca", help: "Bainbridge Island, Central Kitsap, and North Kitsap" },
  { name: "Sammamish Trails", help: "Kirkland, Redmond" },
  { name: "Sinclair", help: "Belfair, Bremerton, Port Orchard and surrounding communities" },
  { name: "Thunderbird", help: "Beacon Hill, Capitol Hill, Central Seattle, South Seattle, Rainier Valley" },
];

export const staffRoles: Array<{ name: string, description?: string }> = [
  { name: "" },
  { name: "instructor", description: "Instructor" },
  { name: "midway", description: "Midway" },
  { name: "support", description: "Support (working for the conference director)" },
];

export const participantMetadata: Props.ObjectMetadata<Participant> = {
  id: { type: Props.stringType, validation: "truthy" },
  youth: { type: Props.booleanType },
  staff: { type: Props.stringType },
  firstName: { type: Props.stringType, validation: "truthy" },
  lastName: { type: Props.stringType, validation: "truthy" },
  suffix: { type: Props.stringType, optional: true },
  nickname: { type: Props.stringType, optional: true },
  email: { type: Props.stringType, validation: "truthy" },
  phone: { type: Props.stringType, validation: "truthy" },
  address: { type: Props.stringType, validation: "truthy" },
  address2: { type: Props.stringType, optional: true },
  city: { type: Props.stringType, validation: "truthy" },
  state: { type: Props.stringType, validation: "truthy", initialValue: () => ("WA") },
  zip: { type: Props.stringType, validation: "truthy" },
  council: { type: Props.stringType, validation: "truthy" },
  district: { type: Props.stringType },
  unitType: { type: Props.stringType, validation: "truthy" },
  unitNumber: { type: Props.stringType },
  bsaNumber: { type: Props.stringType },
  mealRequirements: { type: Props.stringType },
  showQRCode: { type: Props.booleanType, initialValue: () => (true) },
  oaBanquet: { type: Props.booleanType },
  scoutingYears: { type: Props.floatgt0Type, optional: true },
  marketing: { type: Props.stringArrayType },
  marketingOther: { type: Props.stringType, optional: true },
  classes: { type: Props.intArrayType },
  instructorClasses: { type: Props.intArrayType },
  instructorNote: { type: Props.stringType },
};

export function validateParticipant(p: Participant): Props.InvalidProperties<Participant> {
  const invalid = Props.validateObject(participantMetadata, p);

  if (!p.unitNumber && unitTypeHasNumber(p.unitType)) {
    invalid.unitNumber = true;
  }

  if (!p.district && councilHasDistrict(p.council)) {
    invalid.district = true;
  }

  if (p.staff === "instructor"
    && (p.instructorClasses === undefined || p.instructorClasses!.length === 0)
    && !p.instructorNote) {
    invalid.instructorClasses = true;
    invalid.instructorNote = true;
  }

  return invalid;
}

let lastID = 0;

export function newParticipant(): Participant {
  const p = Props.newObject(participantMetadata);
  p.id = (lastID++).toString();
  return p;
}

export function fixupParticipant(p: Participant): Participant {
  if (p.district && !councilHasDistrict(p.council)) {
    p = { ...p };
    p.district = "";
  }
  if (p.unitNumber && !unitTypeHasNumber(p.unitType)) {
    p = { ...p };
    p.unitNumber = "";
  }
  return p;
}

export const registrationMetadata: Props.ObjectMetadata<Registration> = {
  participants: { type: Props.otherType, initialValue: () => ([newParticipant()]) },
};

export function newRegistration(): Registration {
  const r = Props.newObject(registrationMetadata);
  return r;
}

let lastElementID = 0;
export function uniqueID(prefix: string): string {
  lastElementID++;
  return `${prefix}-${lastElementID}`;
}

export let sessionTimes: Array<{ start: string; end: string }> = [
  { start: "9:00", end: "10:00" },
  { start: "10:10", end: "11:10" },
  { start: "11:20", end: "1:15" },
  { start: "1:25", end: "2:25" },
  { start: "2:35", end: "3:35" },
  { start: "3:45", end: "4:45" },
];

export function ClassStartTime(classEntry: Class) : string
{
  if (classEntry.startSession <0 || classEntry.startSession > sessionTimes.length)
  {
    console.log( classEntry.title + " has an invalid start time value");
    return "";
  }
  return sessionTimes[classEntry.startSession].start;
}

export function ClassEndTime(classEntry: Class) : string
{
  if (classEntry.startSession+classEntry.len <0 || classEntry.startSession+classEntry.len > sessionTimes.length)
  {
    console.log( classEntry.title + " has an invalid end time value (", classEntry.startSession,classEntry.len, ")");
    return "";
  }
  return sessionTimes[classEntry.startSession].end;
}