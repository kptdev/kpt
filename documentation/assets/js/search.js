/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

(function($) {

    'use strict';

    function executeSearch($input) {
        var query = ($input.val() || '').trim();
        if (!query) {
            return;
        }

        var searchPage = $input.data('search-page') || '/search/';
        var separator = searchPage.indexOf('?') === -1 ? '?' : '&';
        window.location.href = searchPage + separator + 'q=' + encodeURIComponent(query);
    }

    var Search = {
        init: function() {
            $(document).ready(function() {
                $(document).on('submit', '.td-search-form', function(e) {
                    e.preventDefault();
                    var $input = $(this).find('.td-search-input');
                    executeSearch($input);
                    return false;
                });

                $(document).on('keydown', '.td-search-input', function(e) {
                    if (e.key === 'Enter' || e.keyCode === 13) {
                        e.preventDefault();
                        executeSearch($(this));
                        return false;
                    }
                });

                $(document).on('click', '.td-search-icon-btn', function(e) {
                    var $input = $(this).closest('.add-search-icon, .td-search-form').find('.td-search-input');
                    if ($input.length && ($input.val() || '').trim()) {
                        e.preventDefault();
                        executeSearch($input);
                    }
                });
            });
        },
    };

    Search.init();

}(jQuery));