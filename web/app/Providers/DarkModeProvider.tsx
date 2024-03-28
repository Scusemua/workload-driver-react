import React, { createContext, useState } from 'react';

type ThemeContext = {
    darkMode: boolean;
    toggleDarkMode: () => void;
};

const initialState: ThemeContext = {
    darkMode: true,
    toggleDarkMode: () => {},
};

const DarkModeContext = createContext(initialState);

function DarkModeProvider(props) {
    const root: Element = document.getElementsByTagName('html')[0] as Element;
    const [darkMode, setDarkMode] = useState(initialState.darkMode);

    if (darkMode) {
        root.className = 'pf-v5-theme-dark';
    } else {
        root.className = '';
    }

    const toggleDarkMode = () => {
        const nextMode = !darkMode;
        setDarkMode(!darkMode);

        if (nextMode) {
            root.className = 'pf-v5-theme-dark';
        } else {
            root.className = '';
        }
    };
    return (
        <div>
            <DarkModeContext.Provider value={{ darkMode, toggleDarkMode }}>{props.children}</DarkModeContext.Provider>
        </div>
    );
}

export { DarkModeContext, DarkModeProvider };
