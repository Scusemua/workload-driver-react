import { Monaco } from '@monaco-editor/react';
import { CodeEditor, Language } from '@patternfly/react-code-editor';
import {
    Button,
    Chip,
    Dropdown,
    DropdownItem,
    DropdownList,
    Grid,
    GridItem,
    MenuToggle,
    MenuToggleElement,
    Switch,
    TextInput,
    Tooltip,
} from '@patternfly/react-core';
import { AsleepIcon, DiceIcon, RunningIcon } from '@patternfly/react-icons';
import { GpuIcon, TemplateIcon } from '@src/Assets/Icons';
import { DarkModeContext } from '@src/Providers';
import { editor } from 'monaco-editor/esm/vs/editor/editor.api';
import React, { ReactElement } from 'react';

export interface CodeEditorComponentProps {
    children?: React.ReactNode;
    showCodeTemplates: boolean;
    height: number;
    language: Language;
    targetContext: React.Context<{ code: string; setCode: (_: string) => void }>;

    // Do not include the file extension. That is added automatically.
    defaultFilename: string;
}

interface CodeTemplate {
    name: string;
    code: string;
    icon?: ReactElement;
}

const codeTemplates: CodeTemplate[] = [
    {
        name: 'Declare 1st integer',
        code: `a = 1\nprint("a = %d" % a)`,
        icon: <TemplateIcon />,
    },
    {
        name: 'Increment 1st integer',
        code: 'a = a + 1\nprint("a = %d" % a)',
        icon: <TemplateIcon />,
    },
    {
        name: 'Declare 2nd integer',
        code: `b = a * 2\nprint("a = %d, b = %d" % (a, b))`,
        icon: <TemplateIcon />,
    },
    {
        name: 'Random integer',
        code: `import random\nvar: int = random.randint(0, int(1e6))\nprint(f"Generated a random value: {var}")\nvar = var + 1\nprint(f"Incremented: {var}")`,
        icon: <DiceIcon />,
    },
    {
        name: 'Loop with sleep',
        code: `import time\ncounter: int = 0\nend: int = 10\n\nfor i in range(0, end, 1):\n\tprint(f"i = {i}, counter = {counter}")\n\tcounter = counter + 1\n\ttime.sleep(1)\n\nprint(f"counter={counter}")`,
        icon: <AsleepIcon />,
    },
    {
        name: 'Simulate DL Training',
        code: `# This is the code we run in a notebook cell to simulate training.\nimport socket, os\nsock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)\n\n# Connect to the kernel's TCP socket.\nsock.connect(("127.0.0.1", 5555))\nprint(f'Connected to local TCP server. Local addr: {sock.getsockname()}')\n\n# Blocking call.\n# When training ends, the kernel will be sent a notification.\n# It will then send us a message, unblocking us here and allowing to finish the cell execution.\nsock.recv(1024)\n\nprint("Received 'stop' notification. Done training.")\n\ndel sock`,
        icon: <RunningIcon />,
    },
    {
        name: 'Actual DL Training',
        code: 'training_duration_millis = 1500',
        icon: <GpuIcon />,
    },
];

export const CodeEditorComponent: React.FunctionComponent<CodeEditorComponentProps> = (
    props: CodeEditorComponentProps,
) => {
    const { darkMode } = React.useContext(DarkModeContext);
    const { code, setCode } = React.useContext(props.targetContext);

    const [isCodeTemplateDropdownOpen, setCodeTemplateDropdownOpen] = React.useState<boolean>(false);
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
        <Tooltip
            content={'Filename (for downloading the code)'}
            position={'bottom'}
            key={'template-filename-text-input-tooltip'}
        >
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
        </Tooltip>
    );

    const darkLightThemeSwitch = (
        <div id={'dark-light-theme-switch-container'} key={'dark-light-theme-switch-container'}>
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
                    labelOff="Switch to Dark Theme"
                    isChecked={isEditorDarkMode}
                    onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) => {
                        setIsEditorDarkMode(checked);
                    }}
                />
            </Button>
        </div>
    );

    const onToggleCodeTemplateDropdownClick = () => {
        setCodeTemplateDropdownOpen(!isCodeTemplateDropdownOpen);
    };

    const onSelectCodeTemplate = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined,
    ) => {
        if ((value as number) > codeTemplates.length) {
            console.error(`Invalid code template selected with index=${value}`);
            return;
        }

        const selectedTemplate: CodeTemplate = codeTemplates[value as number];
        if (!selectedTemplate) {
            console.error(`Invalid code template selected with index=${value}`);
            return;
        }

        console.log(`Selected code template #${value}: "${selectedTemplate.name}"`);
        setCodeTemplateDropdownOpen(false);
        setCode(selectedTemplate.code);
    };

    const codeTemplateDropdown = (
        <Dropdown
            key={'code-template-dropdown-menu'}
            id={'code-template-dropdown-menu'}
            isOpen={isCodeTemplateDropdownOpen}
            onSelect={onSelectCodeTemplate}
            onOpenChange={(isOpen: boolean) => setCodeTemplateDropdownOpen(isOpen)}
            toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                <MenuToggle
                    ref={toggleRef}
                    onClick={onToggleCodeTemplateDropdownClick}
                    isExpanded={isCodeTemplateDropdownOpen}
                >
                    Load Template
                </MenuToggle>
            )}
            ouiaId="BasicDropdown"
            shouldFocusToggleOnSelect
        >
            <DropdownList>
                {codeTemplates.map((template: CodeTemplate, idx: number) => {
                    return (
                        <DropdownItem value={idx} key={`code-template-${idx}}`} icon={template.icon}>
                            {template.name}
                        </DropdownItem>
                    );
                })}
            </DropdownList>
        </Dropdown>
    );

    const getCustomControls = () => {
        if (props.showCodeTemplates) {
            return [fileNameField, codeTemplateDropdown, darkLightThemeSwitch];
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
            width={'100%'}
        />
    );
};
