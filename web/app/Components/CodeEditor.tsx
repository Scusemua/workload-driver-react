import React from 'react';
import { CodeEditor, CodeEditorControl, Language } from '@patternfly/react-code-editor';
import { Button, Chip, Grid, GridItem, Switch } from '@patternfly/react-core';
import { editor } from 'monaco-editor/esm/vs/editor/editor.api';
import { CodeIcon } from '@patternfly/react-icons';
import { Monaco } from '@monaco-editor/react';
import { DarkModeContext } from '@app/Providers';
import { CodeContext } from './Modals';

// export interface CodeEditorComponent {
//     children?: React.ReactNode;
// }

export const CodeEditorComponent: React.FunctionComponent = () => {
    const { darkMode } = React.useContext(DarkModeContext);
    const [isEditorDarkMode, setIsEditorDarkMode] = React.useState(darkMode);

    const { code, setCode } = React.useContext(CodeContext);

    const onEditorDidMount = (editor: editor.IStandaloneCodeEditor, monaco: Monaco) => {
        editor.layout();
        editor.focus();
        monaco.editor.getModels()[0].updateOptions({ tabSize: 5 });
    };

    const shortcuts = [
        {
            keys: ['Opt', 'F1'],
            description: 'Accessibility helps',
        },
        {
            keys: ['F1'],
            description: 'View all editor shortcuts',
        },
        {
            keys: ['Ctrl', 'Space'],
            description: 'Activate auto complete',
        },
        {
            keys: ['Cmd', 'S'],
            description: 'Save',
        },
    ];
    const shortcutsPopoverProps = {
        bodyContent: (
            <Grid span={6} hasGutter key="grid">
                {shortcuts.map((shortcut, index) => (
                    <React.Fragment key={index}>
                        <GridItem style={{ textAlign: 'right', marginRight: '1em' }}>
                            {shortcut.keys
                                .map((key) => (
                                    <Chip key={key} isReadOnly>
                                        {key}
                                    </Chip>
                                ))
                                .reduce((prev, curr) => (
                                    <>{[prev, ' + ', curr]}</>
                                ))}
                        </GridItem>
                        <GridItem>{shortcut.description}</GridItem>
                    </React.Fragment>
                ))}
            </Grid>
        ),
        'aria-label': 'Shortcuts',
    };

    const darkLightThemeSwitch = (
        <div>
            <Button
                variant="link"
                onClick={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
                    event.stopPropagation();
                }}
                onMouseDown={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
                    event.preventDefault();
                }}
            >
                <Switch
                    id="code-editor-darkmode-switch"
                    aria-label="darkmode-switch"
                    label="Dark Theme"
                    labelOff="Light Theme"
                    isChecked={isEditorDarkMode}
                    onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) => {
                        setIsEditorDarkMode(checked);
                    }}
                />
            </Button>
        </div>
    );

    const defaultCodeTemplate1 = (
        <CodeEditorControl
            icon={<CodeIcon />}
            aria-label="Execute code"
            tooltipProps={{ content: 'Sample Code #1' }}
            onClick={() => {
                setCode(
                    `a = 15
print("a = %d" % a)`,
                );
            }}
        />
    );

    const defaultCodeTemplate2 = (
        <CodeEditorControl
            icon={<CodeIcon />}
            aria-label="Execute code"
            tooltipProps={{ content: 'Sample Code #2' }}
            onClick={() => {
                setCode(
                    `b = a * 2
print("a = %d, b = %d" % (a, b))`,
                );
            }}
        />
    );

    const defaultCodeTemplate3 = (
        <CodeEditorControl
            icon={<CodeIcon />}
            aria-label="Execute code"
            tooltipProps={{ content: 'Sample Code #3' }}
            onClick={() => {
                setCode(`c = (b + 15) * ((a - 2) * a)
print("a = %d, b = %d, c = %d" % (a, b, c))`);
            }}
        />
    );

    const defaultCodeTemplate4 = (
        <CodeEditorControl
            icon={<CodeIcon />}
            aria-label="Execute code"
            tooltipProps={{ content: 'Sample Code #4' }}
            onClick={() => {
                setCode(`a = 15
b = a + 2
c = (b * 3) - a
d = (2 * a) - (4 * b) + (3 + c)
e = (d ^ 2) + c + b + a
f = e + d + c + b + a
g = f + e + d + c + b + a
h = g + f + e + d + c + b + a
i = h + g + f + e + d + c + b + a
j = i + h + g + f + e + d + c + b + a
print(f"i = {i}")
`);
            }}
        />
    );

    return (
        <CodeEditor
            isDarkTheme={isEditorDarkMode}
            shortcutsPopoverProps={shortcutsPopoverProps}
            customControls={[defaultCodeTemplate1, defaultCodeTemplate2, defaultCodeTemplate3, defaultCodeTemplate4, darkLightThemeSwitch]}
            isLanguageLabelVisible
            isUploadEnabled
            isDownloadEnabled
            isCopyEnabled
            code={code}
            /* eslint-disable-next-line @typescript-eslint/no-unused-vars */
            onChange={(value: string, event: editor.IModelContentChangedEvent) => {
                setCode(value);
            }}
            language={Language.python}
            onEditorDidMount={onEditorDidMount}
            height="400px"
        />
    );
};
