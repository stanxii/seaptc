import * as React from "react";
import * as ReactDOM from "react-dom";
import { BrowserRouter as Router, Route, Link } from "react-router-dom";

import * as Conf from "./conference";

import { RegistrationComponent } from "./RegistrationComponent";
import { HomePageContentComponent } from "./HomePageComponent";
import { ClassEditingComponent } from "./ClassEditingComponent";

var bIsAdminUser = true; // BUGBUG Gary - actually have an auth flow :)
function NewRegistration()
{
    return <RegistrationComponent
        registration={Conf.newRegistration()} />;
}

function Index()
{
    return <div><HomePageContentComponent /></div>;
}

function ExistingRegistration()
{
    return <h2>Edit Registration</h2>;
}

function AppRouter() {
    return (
        <Router>
        <div>
            <nav>
            <ul>
                <li>
                <Link to="/">Home</Link>
                </li>
                <li>
                <Link to="/new/">New Registration</Link>
                </li>
                <li>
                <Link to="/edit/">Edit Existing Registration</Link>
                </li>
                { bIsAdminUser &&
                <div>
                    <li>
                    <Link to="/admin/editclasses/">Edit Classes</Link>
                    </li>
                </div>
                }                
            </ul>
            </nav>

            <Route path="/" exact component={Index} />
            <Route path="/new/" component={NewRegistration} />
            <Route path="/edit/" component={ExistingRegistration} />
            <Route path="/admin/editclasses/" component={ClassEditingComponent} />
        </div>
        </Router>
    );
}


ReactDOM.render(
    <AppRouter />,
    document.getElementById('root')
);