## Scenario

I have a single value replacement in my package, I don’t want package consumers to look through all the yaml files to find the value I want them to set, it seems easier to just create a parameter for this value and have the user look at Kptfile for inputs.

## Problems

1. With popularity the single values inevitably expand to provide a facade to a large portion of the data defeating the purpose of minimizing the cognitive load.
1. Some values like resource names are used as references so setting them in one place needs to trigger updates in all the places where they are referenced.
1. If additional resources that have similar values are added to the package new string replacements need to be added.
1. If a package is used as a sub-package the string replacement parameters need to be surfaced to the parent package and if the parent package already expects some values to be set and the parameters do not exist, the sub-package needs to be updated.

## Suggestions

1. kpt allows the user to edit a particular value directly in the configuration data and will handle upstream merge.  When [editing the yaml] directly the consumers are not confined to the parameters that the package author has 
provided.  kpt has made an investment into upstream merge that allows 
[updating a package] that has been changed with a text editor. )
1. Attributes like resource names which are often updated by consumers to add prefix or suffix (e.g. *-dev, *-stage, *-prod, na1-*, eu1-*) are best handled by the [ensure-name-substring] function that will handle dependency updates as well as capture all the resources in the package.
1. Instead of setting a particular value on a resource a bulk operation can be applied to all the resources that fit a particular interface.  This can be done by a custom function or by [search-and-replace] , [set-labels] and [set-annotations] functions.


[editing the yaml]: /book/03-packages/03-editing-a-package
[updating a package]: /book/03-packages/05-updating-a-package
[ensure-name-substring]: https://catalog.kpt.dev/ensure-name-substring/v0.1/
[search-and-replace]: https://catalog.kpt.dev/search-replace/v0.2/
[set-labels]: https://catalog.kpt.dev/set-labels/v0.1/
[set-annotations]: https://catalog.kpt.dev/set-annotations/v0.1/