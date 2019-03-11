import * as React from "react";

import * as Conf from "./conference";

export interface ClassDetailsProps {
    bDarkRow : boolean;
    classDetails : Conf.Class;
}

export interface ClassDetailsState {
    bEditMode : boolean;
    classDetailsEdited : Conf.Class;

}


class ClassDetailsListEntry extends React.Component<ClassDetailsProps, ClassDetailsState> {

    constructor(props: ClassDetailsProps) {
        super(props);
        // the spread op here forces a copy of the object so we don't just update a ref by mistake
        this.state = { bEditMode: false, classDetailsEdited : {...this.props.classDetails} };
      }
      
    public onEditClassDetails = () : void =>
    {
        this.setState( {bEditMode: true})
    }

    public onSaveClassDetailChanges = () : void =>
    {
        // BUGBUG - api post changes here
        console.log( "BUGBUG - need to post api changes");
        this.props.classDetails.title = this.state.classDetailsEdited.title;
        this.setState( {bEditMode: false})
    }

    public onDiscardClassDetailChanges = () : void =>
    {
        this.setState( {bEditMode: false, classDetailsEdited: this.props.classDetails})
    }

    public OnEditClassTitle = (event : React.FormEvent<HTMLInputElement>) : void =>
    {
        var editedClassDetails = this.state.classDetailsEdited;
        editedClassDetails.title = event.currentTarget.value;
        this.setState( {classDetailsEdited: editedClassDetails})
    }

    public render(): React.ReactNode {
        if(this.state.classDetailsEdited.freeTime)
            return <div/>; // don't put empty sessions in the edit list

        if(this.state.bEditMode)
            return this.renderEditMode();
        else
            return this.renderViewMode();
    }

    public renderViewMode = (): React.ReactNode => {
        var containerClasses = "list-group-item list-group-item-action";
        if(this.props.bDarkRow)
        {
            containerClasses += " list-group-item-dark"
        }
        return <div className={containerClasses}
                    onClick={this.onEditClassDetails}
                   >
            <div className="d-flex w-100 justify-content-between">{this.state.classDetailsEdited.title}
                <small>{Conf.ClassStartTime(this.state.classDetailsEdited)} &ndash; 
                    {Conf.ClassEndTime(this.state.classDetailsEdited)}</small>
            </div>
            <div className="d-flex w-100 justify-content-between">The description is long here dfgdf df gdfgdf{this.state.classDetailsEdited.description}</div>

            </div>;
    }

    public renderEditMode = (): React.ReactNode => {
        var containerClasses = "list-group-item list-group-item-action";
        if(this.props.bDarkRow)
        {
            containerClasses += " list-group-item-dark"
        }
        return <form className={containerClasses}>
            <input className="d-flex w-100 justify-content-between" value={this.state.classDetailsEdited.title}
                onChange={this.OnEditClassTitle} />
            <small>{Conf.ClassStartTime(this.state.classDetailsEdited)} &ndash; 
                    {Conf.ClassEndTime(this.state.classDetailsEdited)}</small>
            <div className="d-flex w-100 justify-content-between">The description is long here dfgdf df gdfgdf{this.state.classDetailsEdited.description}</div>
            <div className="w-100 d-flex justify-content-end">
            <button className="btn btn-sm btn-outline-dark" onClick={this.onSaveClassDetailChanges}>Save</button>&nbsp;
            <button className="btn btn-sm btn-outline-dark" onClick={this.onDiscardClassDetailChanges}>Discard</button>
            </div>
            </form>;
    }
}

export class ClassEditingComponent extends React.Component {
   
    public render(): React.ReactNode {
        var rowNum = -1;
        var classes = Conf.classes.map( (classEntry, idx) => 
            { return <ClassDetailsListEntry classDetails={classEntry} key={classEntry.num} bDarkRow={idx%2==0} />;} )
        return <div className ="ClassEdit list-group">
        {classes}
        </div>;
    }
}