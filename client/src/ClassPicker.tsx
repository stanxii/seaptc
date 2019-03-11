import * as React from "react";
import * as Conf from "./conference";

const classPickerHelp = "Choose your classes. " +
  "There are six class sessions during the day. Some classes " +
  "run more than one session. You cannot select overlapping " +
  "classes. Sold out classes cannot be selected.";

const classPickerInstructorHelp = "Instructors: Check the classes " +
  "you are teaching. If you are teaching a part of a class " +
  "and that class overlaps with a class that you are taking, " +
  "then select the class that you are taking and note the " +
  "class you teaching in the text field below. ";

interface SessionOptions {
  plain: React.ReactNode[];
  withPleaseSelect: React.ReactNode[];
}

let sessionOptionsCache: SessionOptions[];

function getSessionOptions(): SessionOptions[] {
  if (sessionOptionsCache) {
    return sessionOptionsCache;
  }

  const pleaseSelectOption = <option key="pleaseselect" value="" disabled>Please select...</option>;

  const result: SessionOptions[] = [];
  for (let session = 0; session < Conf.numSessions; session++) {
    result[session] = { plain: [], withPleaseSelect: [] };
  }
  for (const cls of Conf.classes) {
    const startSession = cls.startSession;
    const classOption = <option key={cls.num} value={cls.num}>{cls.fullTitle}</option>;
    if (cls.freeTime) {
      const sessionOptions = result[startSession];
      if (sessionOptions.plain.length > 0) {
        const beforeSep = <option key={"b" + cls.num} value="sep" disabled>—</option>;
        sessionOptions.plain.push(beforeSep);
        sessionOptions.withPleaseSelect.push(beforeSep);
      }
      sessionOptions.withPleaseSelect.push(pleaseSelectOption);
      sessionOptions.plain.push(classOption);
      sessionOptions.withPleaseSelect.push(classOption);
      const afterSep = <option key={"a" + cls.num} value="sep" disabled>—</option>;
      sessionOptions.plain.push(afterSep);
      sessionOptions.withPleaseSelect.push(afterSep);
    } else {
      for (let i = 0; i < cls.len; i++) {
        const sessionOptions = result[startSession + i];
        sessionOptions.plain.push(classOption);
        sessionOptions.withPleaseSelect.push(classOption);
      }
    }
  }
  sessionOptionsCache = result;
  return result;
}

export interface ClassPickerProps {
  className: string;
  classes: number[];
  instructor: boolean;
  instructorClasses: number[];
  onChange(classes: number[], instructorClasses: number[]): void;
}

interface ClassPickerUndo {
  deletedClasses: Conf.Class[];
  prevInstructorNums: number[];
  prevNums: number[];
  selectedClass: Conf.Class;
  session: number;
}

interface ClassPickerState {
  undo?: ClassPickerUndo;
}

export class ClassPicker extends React.PureComponent<ClassPickerProps, ClassPickerState> {

  public render(): React.ReactNode {
    const values: Array<string | undefined> = [];
    const instructorChecks: Array<boolean | undefined> = [];

    const set = (nums: number[], instructor: boolean) => {
      for (const num of nums) {
        const cls = Conf.lookupClass(num);
        for (let i = 0; i < cls.len; i++) {
          const session = cls.startSession + i;
          values[session] = num.toString();
          if (!cls.freeTime) {
            instructorChecks[session] = instructor;
          }
        }
      }
    };

    set(this.props.classes || [], false);
    set(this.props.instructorClasses || [], true);

    const undoSession = this.state && this.state.undo
      ? this.state.undo.session
      : undefined;

    const sessionOptions = getSessionOptions();

    const children: React.ReactNode[] = [];
    for (let session = 0; session < Conf.numSessions; session++) {
      const selectID = Conf.uniqueID("session" + session.toString());
      const checkID = selectID + "i";
      children.push(<li key={session} className="list-group-item">
        <div className="row">
          <div className="col-auto">Session {session + 1}<br />
            <small>{Conf.sessionTimes[session].start} &ndash; {Conf.sessionTimes[session].end}</small></div>
          <div className="col">
            <select id={selectID} data-session={session} className="form-control"
              onChange={this.handleClassChange} value={values[session] || ""}>
              {!values[session]
                ? sessionOptions[session].withPleaseSelect
                : sessionOptions[session].plain}
            </select>
            {this.props.instructor && <div key="instructor" className="form-check mt-2">
              <input id={checkID}
                className="form-check-input"
                data-num={values[session]}
                type="checkbox"
                onChange={this.handleInstructorChange}
                disabled={instructorChecks[session] === undefined}
                checked={instructorChecks[session] || false} />
              <label className="form-check-label" htmlFor={checkID}>I am teaching this class.</label>
            </div>}
            {session === undoSession && <UndoNotice
              undo={this.state.undo!}
              classNum={values[session]!}
              onDismiss={this.handleDismissAlert}
              onUndo={this.handleUndo}
            />}
          </div>
        </div>
      </li>);
    }

    return <ul className={`list-group ${this.props.className}`}>
      <p key="help">{classPickerHelp}</p>
      {this.props.instructor && <p key="helpi">{classPickerInstructorHelp}</p>}
      {children}
    </ul>;
  }

  private handleDismissAlert = (e: React.MouseEvent<HTMLButtonElement>) => {
    this.setState({ undo: undefined });
  }

  private handleUndo = (e: React.MouseEvent<HTMLButtonElement>) => {
    this.setState((prevState: ClassPickerState, props: ClassPickerProps): ClassPickerState => {
      if (!prevState.undo) {
        return prevState;
      }
      props.onChange(prevState.undo.prevNums, prevState.undo.prevInstructorNums);
      return { undo: undefined };
    });
  }

  private handleInstructorChange = (ev: React.ChangeEvent<HTMLInputElement>) => {
    const num = parseInt(ev.currentTarget.dataset.num || "", 10);
    let src = this.props.instructorClasses;
    let dst = this.props.classes;
    if (ev.currentTarget.checked) {
      [src, dst] = [dst, src];
    }
    const i = src.indexOf(num);
    if (i < 0) {
      // TODO warn
      return;
    }
    src = [...src];
    src.splice(i, 1);
    dst = [...dst];
    dst.push(num);
    dst.sort((a, b) => (a - b));
    if (ev.currentTarget.checked) {
      [src, dst] = [dst, src];
    }
    this.props.onChange(dst, src);
  }

  private handleClassChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const selectedNum = parseInt(e.currentTarget.value, 10);
    if (selectedNum === undefined) {
      console.log("classpicker could not parse value: ", e.currentTarget.value);
      return;
    }

    const session = parseInt(e.currentTarget.dataset.session || "", 10);
    if (session === undefined) {
      console.log("classpicker could not parse session: ", e.currentTarget.dataset.session);
      return;
    }

    this.setState((prevState: ClassPickerState, props: ClassPickerProps): ClassPickerState => {
      const prevNums = props.classes;
      const prevInstructorNums = props.instructorClasses;
      const nextNums = [selectedNum];
      const nextInstructorNums: number[] = [];

      const selectedClass = Conf.lookupClass(selectedNum);
      const sessionMask = 1 << session;
      let needUndo = false;
      const deletedClasses: Conf.Class[] = [];

      const set = (prev: number[], next: number[]): boolean => {
        for (const num of prev) {
          if (num === selectedNum) {
            return true;
          }
          const cls = Conf.lookupClass(num);
          if ((cls.mask & selectedClass.mask) === 0) {
            next.push(num);
          } else if (!cls.freeTime) {
            deletedClasses.push(cls);
            if ((cls.mask & sessionMask) === 0) {
              needUndo = true;
            }
          }
        }
        return false;
      };

      if (set(prevInstructorNums, nextInstructorNums) || set(prevNums, nextNums)) {
        return prevState;
      }

      nextNums.sort((a, b) => (a - b));
      props.onChange(nextNums, nextInstructorNums);

      if (needUndo) {
        return { undo: { prevNums, prevInstructorNums, deletedClasses, selectedClass, session } };
      } else {
        return { undo: undefined };
      }
    });
  }

}

interface UndoNoticeProps {
  undo: ClassPickerUndo;
  classNum: string;
  onDismiss: (e: React.MouseEvent<HTMLButtonElement>) => void;
  onUndo: (e: React.MouseEvent<HTMLButtonElement>) => void;
}

class UndoNotice extends React.PureComponent<UndoNoticeProps> {
  private ref: React.RefObject<HTMLDivElement>;
  constructor(props: UndoNoticeProps) {
    super(props);
    this.ref = React.createRef();
  }

  public componentDidMount = () => {
    const target = this.ref.current!;
    const rect = target.getBoundingClientRect();
    if (rect.bottom > window.innerHeight) {
        target.scrollIntoView(false);
    } else if (rect.top < 0) {
        target.scrollIntoView();
    }
  }

  public render = (): React.ReactNode => {
    const undo = this.props.undo;
    const deleted = [];
    for (const cls of undo.deletedClasses) {
      deleted.push(<li key={cls.num}>{cls.num}: {cls.title}</li>);
    }
    return <div key="alert" ref={this.ref} className="mt-3">
      <p>The following classes were deleted to make room for class {this.props.classNum}:</p>
      <ul className="mb-3">{...deleted}</ul>
      <button className="btn btn-sm btn-outline-dark mr-2" onClick={this.props.onUndo}>Undo Change</button>
      <button className="btn btn-sm btn-outline-dark" onClick={this.props.onDismiss}>Keep Change</button>
    </div>;
  }
}
