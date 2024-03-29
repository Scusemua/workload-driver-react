import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';
import React from 'react';

interface GpuIconAlt2Props extends SVGIconProps {
    scale?: number;
}

export class GpuIconAlt2 extends React.Component<GpuIconAlt2Props> {
    static displayName = 'GpuIconAlt2';

    id = `icon-title-gpu-alt2-icon`;

    render() {
        let scale = 1;
        if (this.props.scale) {
            scale = this.props.scale;
        }

        const { title, className, ...props } = this.props;
        const classes = className ? `pf-v5-svg ${className}` : 'pf-v5-svg';

        const hasTitle = Boolean(title);
        const viewBox = [0, 0, 59, 59].join(' ');

        return (
            <svg
                className={classes}
                viewBox={viewBox}
                aria-labelledby={hasTitle ? this.id : undefined}
                aria-hidden={hasTitle ? undefined : true}
                role="img"
                width="1em"
                height="1em"
                transform={`scale(${scale})`}
                {...(props as Omit<React.SVGProps<SVGElement>, 'ref'>)} // Lie.
            >
                {hasTitle && <title id={this.id}>{title}</title>}
                <path fill="#38454f" d="M4 12.5h55v32H4z" />
                <circle cx="7" cy="15.5" r="1" fill="#546a79" />
                <circle cx="7" cy="41.5" r="1" fill="#546a79" />
                <circle cx="56" cy="15.5" r="1" fill="#546a79" />
                <circle cx="56" cy="41.5" r="1" fill="#546a79" />
                <path fill="#839594" d="M0 27.5h3v13H0z" />
                <path
                    d="M3 26.5H1c-.553 0-1-.447-1-1s.447-1 1-1h2c.553 0 1 .447 1 1s-.447 1-1 1zm0 17H1c-.553 0-1-.447-1-1s.447-1 1-1h2c.553 0 1 .447 1 1s-.447 1-1 1z"
                    fill="#f3cc6d"
                />
                <path fill="#839594" d="M0 15.5h3v4H0z" />
                <path fill="#28812f" d="M12 44.5h24v4H12z" />
                <path
                    d="M24.389 38.655c-1.76-2.032-2.974-4.996-3.295-8.376-.003-.025-.005-.05-.008-.075-.051-.559-.086-1.125-.086-1.704s.035-1.145.086-1.704c.003-.025.005-.05.008-.075.321-3.38 1.535-6.344 3.295-8.376.781-1.046 1.67-2.005 2.667-2.845H17c-4.971 0-9 5.82-9 13s4.029 13 9 13h10.057c-.998-.84-1.886-1.8-2.668-2.845z"
                    fill="#6c797a"
                />
                <path
                    d="M34.846 41.5C29.534 39.394 26 34.23 26 28.5s3.534-10.894 8.846-13h10.309C50.466 17.606 54 22.77 54 28.5s-3.534 10.894-8.846 13H34.846z"
                    fill="#283238"
                />
                <circle cx="40" cy="28.5" r="3" fill="#cbd4d8" />
                <path
                    d="M49.903 29.739c.119-.499-.359-.91-.848-.753-1.66.535-4.09.448-6.093-.863.016.125.038.248.038.377 0 1.304-.837 2.403-2 2.816 0 0 3.823 2.809 7 3.184.896-1.041 1.557-3.317 1.903-4.761zm-19.884-2.478c-.119.499.359.91.848.753 1.66-.535 4.09-.448 6.093.863-.016-.125-.038-.248-.038-.376 0-1.304.837-2.403 2-2.816 0 0-3.823-2.809-7-3.184-.897 1.04-1.558 3.316-1.903 4.76z"
                    fill="#546a79"
                />
                <path
                    d="M34.343 36.796c.391.333.974.093 1.056-.414.277-1.722 1.457-3.848 3.535-5.037-.118-.043-.238-.079-.353-.137-1.162-.592-1.761-1.837-1.601-3.061 0 0-4.238 2.131-6.015 4.792.52 1.271 2.248 2.894 3.378 3.857zm11.235-16.592c-.391-.333-.974-.093-1.056.414-.277 1.722-1.457 3.848-3.535 5.037.118.043.238.079.353.137 1.162.592 1.761 1.837 1.601 3.061 0 0 4.238-2.131 6.015-4.792-.52-1.271-2.248-2.894-3.378-3.857z"
                    fill="#546a79"
                />
                <path
                    d="M44.179 37.588c.487-.163.582-.787.189-1.118-1.334-1.124-2.548-3.231-2.497-5.624-.097.079-.19.163-.299.232-1.106.691-2.482.563-3.448-.204 0 0-.356 4.73 1.009 7.623 1.357.209 3.638-.437 5.046-.909zm-8.436-18.176c-.487.163-.582.787-.189 1.118 1.334 1.124 2.548 3.231 2.497 5.624.097-.079.19-.163.299-.232 1.106-.691 2.482-.563 3.448.204 0 0 .356-4.73-1.009-7.623-1.358-.209-3.638.437-5.046.909z"
                    fill="#546a79"
                />
                <path
                    d="M14 46.5h2v2h-2zm3 0h2v2h-2zm3 0h2v2h-2zm3 0h2v2h-2zm3 0h2v2h-2zm3 0h2v2h-2zm3 0h2v2h-2z"
                    fill="#f3cc6d"
                />
                <path
                    d="M4 7.5H1c-.553 0-1 .447-1 1s.447 1 1 1h2v41c0 .553.447 1 1 1s1-.447 1-1v-42c0-.553-.447-1-1-1z"
                    fill="#cbd4d8"
                />
            </svg>
        );
    }
}
