import React from 'react';

type ExecutionOutputTabsDataContextType = {
    activeExecutionOutputTab: number;
    setActiveExecutionOutputTab: React.Dispatch<React.SetStateAction<number>>;
    executionOutputTabs: string[];
    setExecutionOutputTabs: React.Dispatch<React.SetStateAction<string[]>>;
    newExecutionOutputTabNumber: number;
    setNewExecutionOutputTabNumber: React.Dispatch<React.SetStateAction<number>>;
    addExecutionOutputTabs: (n: number) => void;
    removeExecutionOutputTabs: (tabIndices: string[] | number[]) => void;
};

const initialState: ExecutionOutputTabsDataContextType = {
    activeExecutionOutputTab: 1,
    executionOutputTabs: ['ExecutionOutput 1'],
    newExecutionOutputTabNumber: 2,
    setActiveExecutionOutputTab: () => {},
    setExecutionOutputTabs: () => {},
    setNewExecutionOutputTabNumber: () => {},
    addExecutionOutputTabs: () => {},
    removeExecutionOutputTabs: () => {},
};

const ExecutionOutputTabsDataContext = React.createContext<ExecutionOutputTabsDataContextType>(initialState);

const ExecutionOutputTabsDataProvider = ({ children }) => {
    const [activeExecutionOutputTab, setActiveExecutionOutputTab] = React.useState<number>(0);
    const [executionOutputTabs, setExecutionOutputTabs] = React.useState<string[]>(['ExecutionOutput 1']);
    const [newExecutionOutputTabNumber, setNewExecutionOutputTabNumber] = React.useState<number>(2);

    const addExecutionOutputTabs = (n: number) => {
        const newTabs: string[] = [];
        let nextExecutionOutputTabNumber: number = newExecutionOutputTabNumber;
        for (let i: number = 0; i < n; i++) {
            newTabs.push(`ExecutionOutput ${nextExecutionOutputTabNumber}`);
            nextExecutionOutputTabNumber += 1;
        }

        setExecutionOutputTabs([...executionOutputTabs, ...newTabs]);
        setNewExecutionOutputTabNumber(nextExecutionOutputTabNumber);
    };

    const removeExecutionOutputTabs = (tabIndices: string[] | number[]) => {
        const indices: number[] = [];
        for (const index of tabIndices) {
            indices.push(index as number);
        }
        indices.sort(function (a, b) {
            return a - b;
        });
        console.log(`Removing executionOutput tab at indices ${JSON.stringify(indices)}`);

        if (indices.length == 0) {
            return;
        }

        // const tabIndexNum = tabIndex as number;
        let nextTabIndex = activeExecutionOutputTab;
        if (indices[indices.length - 1] < activeExecutionOutputTab) {
            // if a preceding tab is closing, keep focus on the new index of the current tab
            nextTabIndex = activeExecutionOutputTab - 1 > 0 ? activeExecutionOutputTab - 1 : 0;
        } else if (activeExecutionOutputTab === executionOutputTabs.length - 1) {
            // if the closing tab is the last tab, focus the preceding tab
            nextTabIndex = executionOutputTabs.length - 2 > 0 ? executionOutputTabs.length - 2 : 0;
        }
        setActiveExecutionOutputTab(nextTabIndex);
        setExecutionOutputTabs(executionOutputTabs.filter((_, index) => !indices.includes(index)));
    };

    return (
        <ExecutionOutputTabsDataContext.Provider
            value={{
                activeExecutionOutputTab,
                setActiveExecutionOutputTab,
                executionOutputTabs,
                setExecutionOutputTabs,
                newExecutionOutputTabNumber,
                setNewExecutionOutputTabNumber,
                addExecutionOutputTabs,
                removeExecutionOutputTabs,
            }}
        >
            {children}
        </ExecutionOutputTabsDataContext.Provider>
    );
};

export { ExecutionOutputTabsDataContext, ExecutionOutputTabsDataProvider, ExecutionOutputTabsDataContextType };
