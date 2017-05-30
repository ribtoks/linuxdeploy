#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <testlib.h>
#include <iostream>

int main(int argc, char *argv[])
{
    QGuiApplication app(argc, argv);

    QQmlApplicationEngine engine;
    QQmlContext *rootContext = engine.rootContext();
    int magicNumber = TestLib().getMagicNumber();
    rootContext->setContextProperty("magicNumber", magicNumber);

    engine.load(QUrl(QStringLiteral("qrc:/main.qml")));

    std::cout << "Loaded main.qml" << std::endl;
    
    return app.exec();
}
