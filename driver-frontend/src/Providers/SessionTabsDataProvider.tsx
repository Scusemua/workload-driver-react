import React from 'react';

type SessionTabsDataContextType = {
    activeSessionTab: number;
    setActiveSessionTab: React.Dispatch<React.SetStateAction<number>>;
    sessionTabs: string[];
    setSessionTabs: React.Dispatch<React.SetStateAction<string[]>>;
    newSessionTabNumber: number;
    setNewSessionTabNumber: React.Dispatch<React.SetStateAction<number>>;
    addSessionTabs: (n: number) => void;
    removeSessionTabs: (tabIndices: string[] | number[]) => void;
};

const initialState: SessionTabsDataContextType = {
    activeSessionTab: 1,
    sessionTabs: ['Session 1'],
    newSessionTabNumber: 2,
    setActiveSessionTab: () => {},
    setSessionTabs: () => {},
    setNewSessionTabNumber: () => {},
    addSessionTabs: () => {},
    removeSessionTabs: () => {},
};

const SessionTabsDataContext = React.createContext<SessionTabsDataContextType>(initialState);

// This exists so we can modify the session tabs within the NewWorkloadFromTemplateModal from any of the components.
// When we load a template from JSON, we need to update the tabs to match the number of sessions from the JSON.
// We do this from the NewWorkloadFromTemplate component. But we modify tabs one-at-a-time when adding or removing
// them from the SessionConfigurationForm component.
const SessionTabsDataProvider = ({ children }) => {
    const [activeSessionTab, setActiveSessionTab] = React.useState<number>(0);
    const [sessionTabs, setSessionTabs] = React.useState<string[]>(['Session 1']);
    const [newSessionTabNumber, setNewSessionTabNumber] = React.useState<number>(2);

    const addSessionTabs = (n: number) => {
        const newTabs: string[] = [];
        let nextSessionTabNumber: number = newSessionTabNumber;
        for (let i: number = 0; i < n; i++) {
            newTabs.push(`Session ${nextSessionTabNumber}`);
            nextSessionTabNumber += 1;
        }

        setSessionTabs([...sessionTabs, ...newTabs]);
        setNewSessionTabNumber(nextSessionTabNumber);
    };

    const removeSessionTabs = (tabIndices: string[] | number[]) => {
        const indices: number[] = [];
        for (const index of tabIndices) {
            indices.push(index as number);
        }
        indices.sort(function (a, b) {
            return a - b;
        });
        console.log(`Removing session tab at indices ${JSON.stringify(indices)}`);

        if (indices.length == 0) {
            return;
        }

        // const tabIndexNum = tabIndex as number;
        let nextTabIndex = activeSessionTab;
        if (indices[indices.length - 1] < activeSessionTab) {
            // if a preceding tab is closing, keep focus on the new index of the current tab
            nextTabIndex = activeSessionTab - 1 > 0 ? activeSessionTab - 1 : 0;
        } else if (activeSessionTab === sessionTabs.length - 1) {
            // if the closing tab is the last tab, focus the preceding tab
            nextTabIndex = sessionTabs.length - 2 > 0 ? sessionTabs.length - 2 : 0;
        }
        setActiveSessionTab(nextTabIndex);
        setSessionTabs(sessionTabs.filter((_, index) => !indices.includes(index)));
    };

    return (
        <SessionTabsDataContext.Provider
            value={{
                activeSessionTab,
                setActiveSessionTab,
                sessionTabs,
                setSessionTabs,
                newSessionTabNumber,
                setNewSessionTabNumber,
                addSessionTabs,
                removeSessionTabs,
            }}
        >
            {children}
        </SessionTabsDataContext.Provider>
    );
};

export { SessionTabsDataContext, SessionTabsDataProvider, SessionTabsDataContextType };
