import { Monaco } from '@monaco-editor/react';
import { CodeEditor, CodeEditorControl, Language } from '@patternfly/react-code-editor';
import { Label, Button, Grid, GridItem, Switch, TextInput } from '@patternfly/react-core';

import { ClockIcon, CodeIcon, DiceIcon } from '@patternfly/react-icons';
import { DarkModeContext } from '@src/Providers';
import { editor } from 'monaco-editor/esm/vs/editor/editor.api';
import React from 'react';
import { CodeContext } from './Modals';

export interface CodeEditorComponentProps {
    children?: React.ReactNode;
    showCodeTemplates: boolean;
    height: number;
    language: Language;

    // Do not include the file extension. That is added automatically.
    defaultFilename: string;
}

export const CodeEditorComponent: React.FunctionComponent<CodeEditorComponentProps> = (
    props: CodeEditorComponentProps,
) => {
    const { darkMode } = React.useContext(DarkModeContext);
    const { code, setCode } = React.useContext(CodeContext);

    const [isEditorDarkMode, setIsEditorDarkMode] = React.useState(darkMode);
    const [filename, setFilename] = React.useState<string>('');

    // If the default filename specified in the props is empty, then use a different default value.
    const defaultFilename: string = props.defaultFilename.length == 0 ? Date.now().toString() : props.defaultFilename;

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
                                    <Label variant="outline" key={key}>
                                        {key}
                                    </Label>
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

    // Function to check if a given filename is valid (for Windows).
    const isValidFilename = (fname: string) => {
        const rg1 = /^[^\\/:*?"<>|]+$/; // forbidden characters \ / : * ? " < > |
        const rg2 = /^\./; // cannot start with dot (.)
        const rg3 = /^(nul|prn|con|lpt[0-9]|com[0-9])(\.|$)/i; // forbidden file names
        return rg1.test(fname) && !rg2.test(fname) && !rg3.test(fname);
    };

    // Function to check if the filename entered by the user.
    // If the filename is empty, we use a default filename, which is a special case insofar
    // as the 'isValidFilename' function returns false for empty strings.
    const isUserFilenameValid = () => {
        if (!filename || filename.length == 0 || isValidFilename(filename)) {
            return 'success';
        }

        return 'error';
    };

    const fileNameField = (
        <TextInput
            key={'template-filename-text-input'}
            // If the user hasn't specified a filename, then don't add the file extension automatically.
            // We'll use the placeholder text instead.
            value={filename}
            label={'Filename'}
            aria-label={'Filename'}
            type="text"
            onChange={(_event, value) => setFilename(value)}
            placeholder={defaultFilename}
            validated={isUserFilenameValid()}
        />
    );

    const darkLightThemeSwitch = (
        <div key={'dark-light-theme-switch-container'}>
            <Button
                key={'dark-light-theme-switch-button-wrapper'}
                variant="link"
                onClick={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
                    event.stopPropagation();
                }}
                onMouseDown={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
                    event.preventDefault();
                }}
            >
                <Switch
                    key="code-editor-darkmode-switch"
                    id="code-editor-darkmode-switch"
                    aria-label="darkmode-switch"
                    label="Switch to Light Theme"
                    isChecked={isEditorDarkMode}
                    onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) => {
                        setIsEditorDarkMode(checked);
                    }}
                />
            </Button>
        </div>
    );

    const defaultCodeTemplate0 = (
        <CodeEditorControl
            icon={<CodeIcon />}
            key={'default-code-template-0'}
            aria-label="Execute code"
            tooltipProps={{ content: 'Sample Code #1' }}
            onClick={() => {
                setCode(
                    `a = 1
print("a = %d" % a)`,
                );
            }}
        />
    );

    const defaultCodeTemplate1 = (
        <CodeEditorControl
            icon={<CodeIcon />}
            key={'default-code-template-1'}
            aria-label="Execute code"
            tooltipProps={{ content: 'Sample Code #1' }}
            onClick={() => {
                setCode(
                    `a = a + 1
print("a = %d" % a)`,
                );
            }}
        />
    );

    const defaultCodeTemplate2 = (
        <CodeEditorControl
            icon={<CodeIcon />}
            key={'default-code-template-2'}
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
            icon={<DiceIcon />}
            key={'default-code-template-3'}
            aria-label="Execute code"
            tooltipProps={{ content: 'Sample Code #3' }}
            onClick={() => {
                setCode(`import random
var: int = random.randint(0, int(1e6))
print(f"Generated a random value: {var}")
var = var + 1
print(f"Incremented: {var}")
`);
            }}
        />
    );

    const defaultCodeTemplate4 = (
        <CodeEditorControl
            icon={<ClockIcon />}
            key={'default-code-template-4'}
            aria-label="Execute code"
            tooltipProps={{ content: 'Sample Code #4' }}
            onClick={() => {
                setCode(`import time
counter: int = 0
end: int = 10

for i in range(0, end, 1):
  print(f"i = {i}, counter = {counter}")
  counter = counter + 1
  time.sleep(1)

print(f"counter={counter}")`);
                //         setCode(`a = 1
                // b = a + 2
                // c = (b * 3) - a
                // d = (2 * a) - (4 * b) + (3 + c)
                // e = (d ^ 2) + c + b + a
                // f = e + d + c + b + a
                // g = f + e + d + c + b + a
                // h = g + f + e + d + c + b + a
                // i = h + g + f + e + d + c + b + a
                // j = i + h + g + f + e + d + c + b + a
                // print(f"i = {i}")
                // `);
            }}
        />
    );

    const getCustomControls = () => {
        if (props.showCodeTemplates) {
            return [
                defaultCodeTemplate0,
                defaultCodeTemplate1,
                defaultCodeTemplate2,
                defaultCodeTemplate3,
                defaultCodeTemplate4,
                fileNameField,
                darkLightThemeSwitch,
            ];
        } else {
            return [fileNameField, darkLightThemeSwitch];
        }
    };

    const getDownloadFilename = () => {
        if (!filename || filename.length == 0) {
            return defaultFilename;
        }

        const fileExtension: string = CodeEditor.getExtensionFromLanguage(props.language);
        if (filename.endsWith(`.${fileExtension}`)) {
            const filenameLength: number = filename.length;
            const extensionLength: number = fileExtension.length + 1; // +1 for the period.
            return filename.substring(0, filenameLength - extensionLength);
        }

        return filename;
    };

    return (
        <CodeEditor
            isDarkTheme={isEditorDarkMode}
            shortcutsPopoverProps={shortcutsPopoverProps}
            customControls={getCustomControls()}
            isLanguageLabelVisible
            isUploadEnabled
            downloadFileName={getDownloadFilename()}
            isDownloadEnabled
            isCopyEnabled
            code={code}
            /* eslint-disable-next-line @typescript-eslint/no-unused-vars */
            onChange={(value: string, _: editor.IModelContentChangedEvent) => {
                setCode(value);
            }}
            onCodeChange={(value: string) => {
                setCode(value);
            }}
            language={props.language}
            onEditorDidMount={onEditorDidMount}
            height={`${props.height}px`}
        />
    );
};
