import * as React from 'react';
import {IntlProvider} from 'react-intl';

import {getTranslations} from 'i18n';

export type Props = {
    currentLocale: string,
    children: React.ReactNode
}

function I18nProvider({currentLocale, children}: Props) {
    return (
        <IntlProvider
            locale={currentLocale}
            key={currentLocale}
            messages={getTranslations(currentLocale)}
        >
            {children}
        </IntlProvider>
    );
}

export default I18nProvider;
