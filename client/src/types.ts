// Types shared with server. TODO: Generate from Go struct.

export interface Participant {
  id: string;
  youth: boolean;
  staff: string;
  firstName: string;
  lastName: string;
  suffix: string;
  nickname: string;
  email: string;
  phone: string;
  address: string;
  address2: string;
  city: string;
  state: string;
  zip: string;
  council: string;
  district: string;
  unitType: string;
  unitNumber: string;
  bsaNumber: string;
  mealRequirements: string;
  showQRCode: boolean;
  oaBanquet: boolean;
  scoutingYears: number;
  marketing: string[];
  marketingOther: string;
  classes: number[];
  instructorClasses: number[];
  instructorNote: string;
}

export interface Registration {
  participants: Participant[];
}

export interface ClassData {
  num: number;
  len: number;
  title: string;
  freeTime?: boolean;
}
