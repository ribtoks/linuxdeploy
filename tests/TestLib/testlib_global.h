#ifndef TESTLIB_GLOBAL_H
#define TESTLIB_GLOBAL_H

#ifdef _WIN32
#  if defined(LIBAVTHUMBNAILER_LIBRARY)
#    define TESTLIBSHARED_EXPORT __declspec(dllexport)
#  else
#    define TESTLIBSHARED_EXPORT __declspec(dllimport)
#  endif
#else
#  define TESTLIBSHARED_EXPORT
#endif

#endif // TESTLIB_GLOBAL_H
