import * as React from "react";

export class HomePageContentComponent extends React.Component {
   
    public render(): React.ReactNode {
        return <div className ="HomePageContent">
            <div className="page-header">
                <h2 itemProp="headline">Program &amp; Training Conference			</h2>
            </div>
            <div itemProp="articleBody">
                <p>The Program and Training Conference (PTC) is a one stop training opportunity for leaders and parents from the entire council to get together and learn more about Scouting. The all day conference features many outstanding opportunities for both position-specific and supplemental training adult leaders in Cub Packs, Scout Troops, Venture Crews and Sea Scouts, as well as district and council committees. It's all at PTC and as you can see in the name... it's about Program and it's about Training which will create a better Scouting program for the youth we serve.</p>
                <dl>
                    <dt>When</dt>
                    <dd>Saturday, October 19th from 7:40 AM to 4:45 PM</dd>
                    <dt>Where</dt>
                    <dd><a href="#where">North Seattle College</a></dd>
                    <dt>Cost</dt>
                    <dd>
                        <ul>
                            <li>$45 for adults ($10 early discount before 10/14/2019)</li>
                            <li>$20 for youth (under age 21)</li>
                            <li>$75 for Wilderness &amp; Remote First Aid (including youth)</li>
                        </ul>
                        <p>The registration fee includes classes, lunch, patch, and resource materials.</p>
                    </dd>
                </dl>
            </div>
        </div>;
    }
}